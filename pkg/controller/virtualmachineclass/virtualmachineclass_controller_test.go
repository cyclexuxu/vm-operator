// +build integration

/* **********************************************************
 * Copyright 2019 VMware, Inc.  All rights reserved. -- VMware Confidential
 * **********************************************************/

package virtualmachineclass

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vmoperatorv1alpha1 "github.com/vmware-tanzu/vm-operator/pkg/apis/vmoperator/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var c client.Client

const timeout = time.Second * 5

var _ = Describe("VirtualMachineClass controller", func() {
	name := "fooVm"

	var (
		instance   vmoperatorv1alpha1.VirtualMachineClass
		invalid    vmoperatorv1alpha1.VirtualMachineClass
		stopMgr    chan struct{}
		mgrStopped *sync.WaitGroup
		mgr        manager.Manager
		err        error
	)

	BeforeEach(func() {
		// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
		// channel when it is finished.
		mgr, err = manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()

		stopMgr, mgrStopped = StartTestManager(mgr)
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()
	})

	Describe("when creating/deleting a VM Class", func() {
		It("invoke the validate method", func() {
			// Create the VM Class object and expect this to fail
			invalid = vmoperatorv1alpha1.VirtualMachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineClassSpec{
					Hardware: vmoperatorv1alpha1.VirtualMachineClassHardware{
						Cpus:   4,
						Memory: resource.MustParse("1Mi"),
					},
					Policies: vmoperatorv1alpha1.VirtualMachineClassPolicies{
						Resources: vmoperatorv1alpha1.VirtualMachineClassResources{
							Requests: vmoperatorv1alpha1.VirtualMachineClassResourceSpec{
								Cpu:    resource.MustParse("2000Mi"),
								Memory: resource.MustParse("100Mi"),
							},
							Limits: vmoperatorv1alpha1.VirtualMachineClassResourceSpec{
								Cpu:    resource.MustParse("1000Mi"),
								Memory: resource.MustParse("200Mi"),
							},
						},
						StorageClass: "fooStorageClass",
					},
				},
			}

			err = c.Create(context.TODO(), &invalid)
			Expect(err).To(HaveOccurred())

			err = c.Delete(context.TODO(), &invalid)
			Expect(err).To(HaveOccurred())
		})

		It("invoke the reconcile method", func() {
			instance = vmoperatorv1alpha1.VirtualMachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineClassSpec{
					Hardware: vmoperatorv1alpha1.VirtualMachineClassHardware{
						Cpus:   4,
						Memory: resource.MustParse("1Mi"),
					},
					Policies: vmoperatorv1alpha1.VirtualMachineClassPolicies{
						Resources: vmoperatorv1alpha1.VirtualMachineClassResources{
							Requests: vmoperatorv1alpha1.VirtualMachineClassResourceSpec{
								Cpu:    resource.MustParse("1000Mi"),
								Memory: resource.MustParse("100Mi"),
							},
							Limits: vmoperatorv1alpha1.VirtualMachineClassResourceSpec{
								Cpu:    resource.MustParse("2000Mi"),
								Memory: resource.MustParse("200Mi"),
							},
						},
						StorageClass: "fooStorageClass",
					},
				},
			}

			expectedRequest := reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}
			recFn, requests := SetupTestReconcile(newReconciler(mgr))
			Expect(add(mgr, recFn)).To(Succeed())
			// Create the VM Class object and expect the Reconcile
			err = c.Create(context.TODO(), &instance)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(requests, timeout).Should(Receive(Equal(expectedRequest)))
			// Delete the VM Class object and expect the Reconcile
			err = c.Delete(context.TODO(), &instance)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(requests, timeout).Should(Receive(Equal(expectedRequest)))
		})
	})
})