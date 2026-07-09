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

	Describe("WatchEnvironmentsFromVersion", func() {
		It("does not miss events that occurred between Get and Watch", func() {
			env := &aliecsv1alpha1.Environment{
				ObjectMeta: metav1.ObjectMeta{Name: "versioned-watch-env"},
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

			got, err := c.GetEnvironment(ctx, env.Name)
			Expect(err).NotTo(HaveOccurred())
			resourceVersionBeforeUpdate := got.ResourceVersion

			got.Spec.State = "configured"
			Expect(c.UpdateEnvironment(ctx, got)).To(Succeed())

			watcher, err := c.WatchEnvironmentsFromVersion(ctx, resourceVersionBeforeUpdate)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			Eventually(watcher.ResultChan()).Should(Receive(Satisfy(func(e k8swatch.Event) bool {
				ev, ok := e.Object.(*aliecsv1alpha1.Environment)
				return ok && ev.Name == env.Name && ev.Spec.State == "configured"
			})))
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

	Describe("ListTasksByLabel", func() {
		newTask := func(name string, labels map[string]string) *aliecsv1alpha1.Task {
			return &aliecsv1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
				Spec:       aliecsv1alpha1.TaskSpec{State: "standby", Pod: v1.PodSpec{Containers: []v1.Container{}}},
			}
		}

		It("returns an error when no tasks match", func() {
			_, err := c.ListTasksByLabel(ctx, map[string]string{"env": "nonexistent"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no Task CRDs found"))
		})

		It("returns tasks matching a single label", func() {
			task := newTask("label-task-1", map[string]string{"env": "test"})
			Expect(c.CreateTask(ctx, task)).To(Succeed())
			defer c.DeleteTask(ctx, task.Name)

			tasks, err := c.ListTasksByLabel(ctx, map[string]string{"env": "test"})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].Name).To(Equal("label-task-1"))
		})

		It("returns multiple tasks matching a label", func() {
			task1 := newTask("label-task-2a", map[string]string{"env": "multi"})
			task2 := newTask("label-task-2b", map[string]string{"env": "multi"})
			Expect(c.CreateTask(ctx, task1)).To(Succeed())
			defer c.DeleteTask(ctx, task1.Name)
			Expect(c.CreateTask(ctx, task2)).To(Succeed())
			defer c.DeleteTask(ctx, task2.Name)

			tasks, err := c.ListTasksByLabel(ctx, map[string]string{"env": "multi"})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(2))
			names := []string{tasks[0].Name, tasks[1].Name}
			Expect(names).To(ConsistOf("label-task-2a", "label-task-2b"))
		})

		It("returns only tasks matching all labels", func() {
			taskBoth := newTask("label-task-3a", map[string]string{"env": "filter", "role": "worker"})
			taskOne := newTask("label-task-3b", map[string]string{"env": "filter"})
			Expect(c.CreateTask(ctx, taskBoth)).To(Succeed())
			defer c.DeleteTask(ctx, taskBoth.Name)
			Expect(c.CreateTask(ctx, taskOne)).To(Succeed())
			defer c.DeleteTask(ctx, taskOne.Name)

			tasks, err := c.ListTasksByLabel(ctx, map[string]string{"env": "filter", "role": "worker"})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].Name).To(Equal("label-task-3a"))
		})

		It("does not return tasks with a different label value", func() {
			taskA := newTask("label-task-4a", map[string]string{"env": "staging"})
			taskB := newTask("label-task-4b", map[string]string{"env": "production"})
			Expect(c.CreateTask(ctx, taskA)).To(Succeed())
			defer c.DeleteTask(ctx, taskA.Name)
			Expect(c.CreateTask(ctx, taskB)).To(Succeed())
			defer c.DeleteTask(ctx, taskB.Name)

			tasks, err := c.ListTasksByLabel(ctx, map[string]string{"env": "staging"})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].Name).To(Equal("label-task-4a"))
		})
	})
})
