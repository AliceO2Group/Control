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

package client

import (
	"context"
	"fmt"

	"github.com/AliceO2Group/Control/control-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	client    crClient.WithWatch
	namespace string
}

func New(kubeconfigPath, namespace string) (*Client, error) {
	config, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}
	return NewFromConfig(config, namespace)
}

func NewFromConfig(config *rest.Config, namespace string) (*Client, error) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("registering v1alpha1 scheme: %w", err)
	}

	c, err := crClient.NewWithWatch(config, crClient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return &Client{client: c, namespace: namespace}, nil
}

func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	return nil, fmt.Errorf("ECS isn't running in a container and you didn't pass any kubernetes config: %v", err)
}

func (c *Client) CreateTask(ctx context.Context, task *v1alpha1.Task) error {
	task.Namespace = c.namespace
	return c.client.Create(ctx, task)
}

func (c *Client) GetTask(ctx context.Context, name string) (*v1alpha1.Task, error) {
	task := &v1alpha1.Task{}
	err := c.client.Get(ctx, types.NamespacedName{Name: name, Namespace: c.namespace}, task)
	return task, err
}

func (c *Client) UpdateTask(ctx context.Context, task *v1alpha1.Task) error {
	task.Namespace = c.namespace
	return c.client.Update(ctx, task)
}

func (c *Client) DeleteTask(ctx context.Context, name string) error {
	return c.client.Delete(ctx, &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: c.namespace},
	})
}

// WatchTasks returns a watcher for all Task resources in the namespace.
// Each event on ResultChan() carries a *v1alpha1.Task as event.Object.
func (c *Client) WatchTasks(ctx context.Context) (watch.Interface, error) {
	return c.client.Watch(ctx, &v1alpha1.TaskList{}, crClient.InNamespace(c.namespace))
}

// ListTasksByLabel returns all Task resources matching the given label selector.
// Returns an error if no tasks are found.
func (c *Client) ListTasksByLabel(ctx context.Context, labels map[string]string) ([]v1alpha1.Task, error) {
	taskList := &v1alpha1.TaskList{}
	if err := c.client.List(ctx, taskList, crClient.InNamespace(c.namespace), crClient.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	if len(taskList.Items) == 0 {
		return nil, fmt.Errorf("no Task CRDs found matching labels %v", labels)
	}
	return taskList.Items, nil
}

func (c *Client) CreateEnvironment(ctx context.Context, env *v1alpha1.Environment) error {
	env.Namespace = c.namespace
	return c.client.Create(ctx, env)
}

func (c *Client) GetEnvironment(ctx context.Context, name string) (*v1alpha1.Environment, error) {
	env := &v1alpha1.Environment{}
	err := c.client.Get(ctx, types.NamespacedName{Name: name, Namespace: c.namespace}, env)
	return env, err
}

func (c *Client) UpdateEnvironment(ctx context.Context, env *v1alpha1.Environment) error {
	env.Namespace = c.namespace
	return c.client.Update(ctx, env)
}

func (c *Client) DeleteEnvironment(ctx context.Context, name string) error {
	return c.client.Delete(ctx, &v1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: c.namespace},
	})
}

// WatchEnvironments returns a watcher for all Environment resources in the namespace.
// Each event on ResultChan() carries a *v1alpha1.Environment as event.Object.
func (c *Client) WatchEnvironments(ctx context.Context) (watch.Interface, error) {
	return c.client.Watch(ctx, &v1alpha1.EnvironmentList{}, crClient.InNamespace(c.namespace))
}

// WatchEnvironmentsFromVersion watches Environment resources starting from the given resourceVersion,
// ensuring no events that occurred after that version are missed.
func (c *Client) WatchEnvironmentsFromVersion(ctx context.Context, resourceVersion string) (watch.Interface, error) {
	return c.client.Watch(ctx, &v1alpha1.EnvironmentList{},
		crClient.InNamespace(c.namespace),
		&crClient.ListOptions{Raw: &metav1.ListOptions{ResourceVersion: resourceVersion}})
}
