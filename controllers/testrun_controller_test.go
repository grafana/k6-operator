package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/grafana/k6-operator/api/v1alpha1"
)

var testRunSuiteReconciler *TestRunReconciler

var _ = Describe("TestRun", func() {
	ctx := context.Background()

	testRun := &v1alpha1.TestRun{}
	starterJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "some-test-starter",
		},
	}

	BeforeEach(func() {
		testRun = &v1alpha1.TestRun{
			ObjectMeta: metav1.ObjectMeta{
				Labels:    map[string]string{k6CrLabelName: "some-test"},
				Name:      "some-test",
				Namespace: "default",
			},
		}
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, testRun)).Error().ToNot(HaveOccurred())
		err := k8sClient.Delete(ctx, starterJob)
		Expect(client.IgnoreNotFound(err)).Error().ToNot(HaveOccurred())
	})

	When("Reconciling a TestRun that is in 'created' stage and spec.paused is set to 'true'", func() {
		It("should prevent the starter job from running", func() {

			testRun.Spec.Paused = "true"
			Expect(k8sClient.Create(ctx, testRun)).Error().ToNot(HaveOccurred())
			testRun.Status.Stage = "created"
			Expect(k8sClient.Status().Update(ctx, testRun)).Error().ToNot(HaveOccurred())

			By("returning no error and no requeue when reconciled", func() {
				result, err := testRunSuiteReconciler.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "some-test",
					},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeZero())
			})

			By("not having started the jobs", func() {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(starterJob), &batchv1.Job{})
				Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	When("Reconciling a TestRun that is in 'created' stage and spec.paused isn't set", func() {
		It("should prevent the starter job from running", func() {

			testRun.Spec.Paused = ""
			Expect(k8sClient.Create(ctx, testRun)).Error().ToNot(HaveOccurred())
			testRun.Status.Stage = "created"
			Expect(k8sClient.Status().Update(ctx, testRun)).Error().ToNot(HaveOccurred())

			By("returning no error and no requeue when reconciled", func() {
				result, err := testRunSuiteReconciler.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "some-test",
					},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeZero())
			})

			By("not having started the jobs", func() {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(starterJob), &batchv1.Job{})
				Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	When("Reconciling a TestRun that is in 'created' stage and spec.paused is set to 'false'", func() {
		It("should create the starter job", func() {

			testRun.Spec.Paused = "false"
			Expect(k8sClient.Create(ctx, testRun)).Error().ToNot(HaveOccurred())
			testRun.Status.Stage = "created"
			Expect(k8sClient.Status().Update(ctx, testRun)).Error().ToNot(HaveOccurred())

			By("returning no error and no requeue when reconciled", func() {
				// we don't care about the result itself for what this test asserts, just the error
				_, err := testRunSuiteReconciler.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "some-test",
					},
				})
				Expect(err).ToNot(HaveOccurred())
			})

			By("having started the jobs", func() {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(starterJob), &batchv1.Job{})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
