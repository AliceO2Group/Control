/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2026 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package client_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swatch "k8s.io/apimachinery/pkg/watch"

	aliecsv1alpha1 "github.com/AliceO2Group/Control/control-operator/api/v1alpha1"
	"github.com/AliceO2Group/Control/control-operator/pkg/client"
)

var _ = Describe("Client", func() {
	var (
		ctx context.Context
		c   *client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		c, err = client.NewFromConfig(cfg, "default")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Task Create, Read, Update, Delete", func() {
		var task *aliecsv1alpha1.Task

		BeforeEach(func() {
			task = &aliecsv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{Name: "test-task"},
				Spec:       aliecsv1alpha1.TaskSpec{State: "standby", Pod: v1.PodSpec{Containers: []v1.Container{}}},
			}
			Expect(c.CreateTask(ctx, task)).To(Succeed())
		})

		AfterEach(func() {
			_ = c.DeleteTask(ctx, task.Name)
		})

		It("gets a created task", func() {
			got, err := c.GetTask(ctx, "test-task")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("test-task"))
			Expect(got.Spec.State).To(Equal("standby"))
		})

		It("updates a task", func() {
			got, err := c.GetTask(ctx, "test-task")
			Expect(err).NotTo(HaveOccurred())

			got.Spec.State = "running"
			Expect(c.UpdateTask(ctx, got)).To(Succeed())

			updated, err := c.GetTask(ctx, "test-task")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Spec.State).To(Equal("running"))
		})

		It("deletes a task", func() {
			Expect(c.DeleteTask(ctx, "test-task")).To(Succeed())

			_, err := c.GetTask(ctx, "test-task")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Task Watch", func() {
		It("receives events for task changes", func() {
			watcher, err := c.WatchTasks(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			task := &aliecsv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{Name: "watch-task"},
				Spec:       aliecsv1alpha1.TaskSpec{State: "standby", Pod: v1.PodSpec{Containers: []v1.Container{}}},
			}
			Expect(c.CreateTask(ctx, task)).To(Succeed())
			defer c.DeleteTask(ctx, task.Name)

			Eventually(watcher.ResultChan()).Should(Receive(Satisfy(func(e k8swatch.Event) bool {
				t, ok := e.Object.(*aliecsv1alpha1.Task)
				return ok && t.Name == "watch-task" && e.Type == k8swatch.Added
			})))
		})
	})

	Describe("Environment Create, Read, Update, Delete", func() {
		var env *aliecsv1alpha1.Environment

		BeforeEach(func() {
			env = &aliecsv1alpha1.Environment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-env"},
				Spec: aliecsv1alpha1.EnvironmentSpec{
					State: "standby",
					Tasks: map[string][]aliecsv1alpha1.TaskDefinition{},
				},
				TaskTemplates: aliecsv1alpha1.TemplateSpecification{
					Tasks: map[string][]aliecsv1alpha1.TaskReference{},
				},
			}
			Expect(c.CreateEnvironment(ctx, env)).To(Succeed())
		})

		AfterEach(func() {
			_ = c.DeleteEnvironment(ctx, env.Name)
		})

		It("gets a created environment", func() {
			got, err := c.GetEnvironment(ctx, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("test-env"))
			Expect(got.Spec.State).To(Equal("standby"))
		})

		It("updates an environment", func() {
			got, err := c.GetEnvironment(ctx, "test-env")
			Expect(err).NotTo(HaveOccurred())

			got.Spec.State = "running"
			Expect(c.UpdateEnvironment(ctx, got)).To(Succeed())

			updated, err := c.GetEnvironment(ctx, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Spec.State).To(Equal("running"))
		})

		It("deletes an environment", func() {
			Expect(c.DeleteEnvironment(ctx, "test-env")).To(Succeed())

			_, err := c.GetEnvironment(ctx, "test-env")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Environment Watch", func() {
		It("receives events for environment changes", func() {
			watcher, err := c.WatchEnvironments(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			env := &aliecsv1alpha1.Environment{
				ObjectMeta: metav1.ObjectMeta{Name: "watch-env"},
				Spec: aliecsv1alpha1.EnvironmentSpec{
					State: "standby",
					Tasks: map[string][]aliecsv1alpha1.TaskDefinition{},
				},
				TaskTemplates: aliecsv1alpha1.TemplateSpecification{
					Tasks: map[string][]aliecsv1alpha1.TaskReference{},
				},
			}
			Expect(c.CreateEnvironment(ctx, env)).To(Succeed())
			defer c.DeleteEnvironment(ctx, env.Name)

			Eventually(watcher.ResultChan()).Should(Receive(Satisfy(func(e k8swatch.Event) bool {
				ev, ok := e.Object.(*aliecsv1alpha1.Environment)
				return ok && ev.Name == "watch-env" && e.Type == k8swatch.Added
			})))
		})
	})
})
