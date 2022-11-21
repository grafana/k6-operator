package controllers

import (
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const k6CrLabelName = "k6_cr"

type K6PodsWatchMap struct {
	log logr.Logger
}

// Map Watch map function used above.
// Obj is a Pod that just got an event, map it back to any matching Migrators.
func (m *K6PodsWatchMap) Map(object handler.MapObject) []reconcile.Request {
	pod := object.Object.(*v1.Pod)
	k6CrName, _ := pod.GetLabels()[k6CrLabelName]
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      k6CrName,
			Namespace: object.Meta.GetNamespace(),
		}}}
}

func filterK6Pods() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			pod := e.Object.(*v1.Pod)
			_, ok := pod.GetLabels()[k6CrLabelName]
			if !ok {
				return false
			}
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			pod := e.MetaNew.(*v1.Pod)
			_, ok := pod.GetLabels()[k6CrLabelName]
			if !ok {
				return false
			}
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			pod := e.Object.(*v1.Pod)
			_, ok := pod.GetLabels()[k6CrLabelName]
			if !ok {
				return false
			}

			return !e.DeleteStateUnknown
		},
	}
}
