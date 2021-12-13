package controllers

import (
	"context"
	"errors"
	"fmt"

	coilv2 "github.com/cybozu-go/coil/v2/api/v2"
	"github.com/cybozu-go/coil/v2/pkg/constants"
	"github.com/cybozu-go/coil/v2/pkg/ipam"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BlockRequestReconciler reconciles a BlockRequest object
type BlockRequestReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Manager ipam.PoolManager
}

// +kubebuilder:rbac:groups=coil.cybozu.com,resources=blockrequests,verbs=get;list;watch
// +kubebuilder:rbac:groups=coil.cybozu.com,resources=blockrequests/status,verbs=get;update;patch

// Reconcile implements Reconciler interface.
// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile?tab=doc#Reconciler
func (r *BlockRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	br := &coilv2.BlockRequest{}
	err := r.Client.Get(ctx, req.NamespacedName, br)

	if err != nil {
		// as Delete event is ignored, this is unlikely to happen.
		logger.Error(err, "failed to get")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if len(br.Status.Conditions) > 0 {
		// as this case is excluded by the event filter, this should not happen.
		return ctrl.Result{}, nil
	}

	blocks := &coilv2.AddressBlockList{}
	err = r.Client.List(ctx, blocks, client.MatchingFields{
		constants.AddressBlockRequestKey: string(br.UID),
	})
	if err != nil {
		logger.Error(err, "failed to list AddressBlock")
		return ctrl.Result{}, nil
	}
	if len(blocks.Items) > 0 {
		if err := r.updateStatus(ctx, br, blocks.Items[0].Name); err != nil {
			logger.Error(err, "a block for the request has been created, but failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	block, err := r.Manager.AllocateBlock(ctx, br.Spec.PoolName, br.Spec.NodeName, string(br.UID))
	if errors.Is(err, ipam.ErrNoBlock) {
		logger.Error(err, "out of blocks", "pool", br.Spec.PoolName)

		now := metav1.Now()
		br.Status.Conditions = []coilv2.BlockRequestCondition{
			{
				Type:               coilv2.BlockRequestComplete,
				Status:             corev1.ConditionTrue,
				Reason:             "completed with failure",
				Message:            "completed with failure",
				LastProbeTime:      now,
				LastTransitionTime: now,
			},
			{
				Type:               coilv2.BlockRequestFailed,
				Status:             corev1.ConditionTrue,
				Reason:             "out of blocks",
				Message:            fmt.Sprintf("pool %s does not have free blocks", br.Spec.PoolName),
				LastProbeTime:      now,
				LastTransitionTime: now,
			},
		}
		err = r.Client.Status().Update(ctx, br)
		if err != nil {
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "internal error")
		return ctrl.Result{}, err
	}

	logger.Info("allocated", "block", block.Name, "index", block.Index, "pool", br.Spec.PoolName)

	if err := r.updateStatus(ctx, br, block.Name); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BlockRequestReconciler) updateStatus(ctx context.Context, br *coilv2.BlockRequest, blockName string) error {
	now := metav1.Now()
	br.Status.Conditions = []coilv2.BlockRequestCondition{
		{
			Type:               coilv2.BlockRequestComplete,
			Status:             corev1.ConditionTrue,
			Reason:             "allocated",
			Message:            fmt.Sprintf("allocated a block %s", blockName),
			LastProbeTime:      now,
			LastTransitionTime: now,
		},
	}
	br.Status.AddressBlockName = blockName
	err := r.Client.Status().Update(ctx, br)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager registers this with the manager.
func (r *BlockRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coilv2.BlockRequest{}, builder.WithPredicates(predicate.Funcs{
			// predicate.Funcs returns true by default
			UpdateFunc: func(ev event.UpdateEvent) bool {
				req := ev.ObjectNew.(*coilv2.BlockRequest)
				return len(req.Status.Conditions) == 0
			},
			DeleteFunc: func(event.DeleteEvent) bool {
				return false
			},
		})).
		Complete(r)
}
