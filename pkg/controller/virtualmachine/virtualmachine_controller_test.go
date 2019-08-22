// +build integration

/* **********************************************************
 * Copyright 2019 VMware, Inc.  All rights reserved. -- VMware Confidential
 * **********************************************************/

package virtualmachine

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
	"github.com/vmware-tanzu/vm-operator/test/integration"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var c client.Client

const timeout = time.Second * 5

var _ = Describe("VirtualMachine controller", func() {
	ns := integration.DefaultNamespace
	name := "fooVm"

	var (
		classInstance   vmoperatorv1alpha1.VirtualMachineClass
		instance        vmoperatorv1alpha1.VirtualMachine
		expectedRequest reconcile.Request
		recFn           reconcile.Reconciler
		requests        chan reconcile.Request
		stopMgr         chan struct{}
		mgrStopped      *sync.WaitGroup
		mgr             manager.Manager
		err             error
	)

	BeforeEach(func() {
		classInstance = vmoperatorv1alpha1.VirtualMachineClass{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
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

		expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Namespace: ns, Name: name}}

		// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
		// channel when it is finished.
		mgr, err = manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()

		recFn, requests = SetupTestReconcile(newReconciler(mgr))
		Expect(add(mgr, recFn)).To(Succeed())

		stopMgr, mgrStopped = StartTestManager(mgr)

		err = c.Create(context.TODO(), &classInstance)
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()
	})

	Describe("when creating/deleting a VM object", func() {
		It("invoke the reconcile method", func() {
			imageList := &vmoperatorv1alpha1.VirtualMachineImageList{}
			err = c.List(context.TODO(), &client.ListOptions{Namespace: ns}, imageList)
			/* TODO() This List call does not seem to not pass along the namespace
			Expect(err).ShouldNot(HaveOccurred())
			imageName := imageList.Items[0].Name
			*/
			imageName := "DC0_H0_VM0"
			instance = vmoperatorv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      name,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineSpec{
					ImageName:  imageName,
					ClassName:  classInstance.Name,
					PowerState: "poweredOn",
					Ports:      []vmoperatorv1alpha1.VirtualMachinePort{},
				},
			}
			// Create the VM Object the expect Reconcile
			err = c.Create(context.TODO(), &instance)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(requests, timeout).Should(Receive(Equal(expectedRequest)))
			// Delete the VM Object the expect Reconcile
			err = c.Delete(context.TODO(), &instance)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(requests, timeout).Should(Receive(Equal(expectedRequest)))
		})
	})
})