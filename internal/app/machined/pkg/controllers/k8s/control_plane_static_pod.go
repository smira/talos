// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package k8s

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/talos-systems/os-runtime/pkg/controller"
	"github.com/talos-systems/os-runtime/pkg/resource"
	"github.com/talos-systems/os-runtime/pkg/state"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/talos-systems/talos/internal/app/machined/pkg/resources/config"
	"github.com/talos-systems/talos/internal/app/machined/pkg/resources/k8s"
	"github.com/talos-systems/talos/pkg/machinery/config/types/v1alpha1/machine"
)

// ControlPlaneStaticPodController manages k8s.StaticPod based on control plane configuration.
type ControlPlaneStaticPodController struct {
}

// Name implements controller.Controller interface.
func (ctrl *ControlPlaneStaticPodController) Name() string {
	return "k8s.ControlPlaneStaticPodController"
}

// ManagedResources implements controller.Controller interface.
func (ctrl *ControlPlaneStaticPodController) ManagedResources() (resource.Namespace, resource.Type) {
	return k8s.ControlPlaneNamespaceName, k8s.StaticPodType
}

// Run implements controller.Controller interface.
//
//nolint: gocyclo
func (ctrl *ControlPlaneStaticPodController) Run(ctx context.Context, r controller.Runtime, logger *log.Logger) error {
	if err := r.UpdateDependencies([]controller.Dependency{
		{
			Namespace: config.NamespaceName,
			Type:      config.K8sControlPlaneType,
			Kind:      controller.DependencyWeak,
		},
		{
			Namespace: config.NamespaceName,
			Type:      config.MachineTypeType,
			ID:        pointer.ToString(config.MachineTypeID),
			Kind:      controller.DependencyWeak,
		},
	}); err != nil {
		return fmt.Errorf("error setting up dependencies: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		machineTypeRes, err := r.Get(ctx, resource.NewMetadata(config.NamespaceName, config.MachineTypeType, config.MachineTypeID, resource.VersionUndefined))
		if err != nil {
			if state.IsNotFoundError(err) {
				continue
			}

			return fmt.Errorf("error getting machine type: %w", err)
		}

		machineType := machineTypeRes.(*config.MachineType).MachineType()

		if machineType != machine.TypeControlPlane && machineType != machine.TypeInit {
			if err = ctrl.teardownAll(ctx, r); err != nil {
				return fmt.Errorf("error destroying static pods: %w", err)
			}
		}

		for _, pod := range []struct {
			f  func(context.Context, controller.Runtime, *log.Logger, *config.K8sControlPlane) error
			id resource.ID
		}{
			{
				f:  ctrl.manageAPIServer,
				id: config.K8sControlPlaneAPIServerID,
			},
		} {
			res, err := r.Get(ctx, resource.NewMetadata(config.NamespaceName, config.K8sControlPlaneType, pod.id, resource.VersionUndefined))
			if err != nil {
				if state.IsNotFoundError(err) {
					continue
				}

				return fmt.Errorf("error getting control plane config: %w", err)
			}

			if err = pod.f(ctx, r, logger, res.(*config.K8sControlPlane)); err != nil {
				return fmt.Errorf("error updating static pod for %q: %w", pod.id, err)
			}
		}
	}
}

func (ctrl *ControlPlaneStaticPodController) teardownAll(ctx context.Context, r controller.Runtime) error {
	list, err := r.List(ctx, resource.NewMetadata(k8s.ControlPlaneNamespaceName, k8s.StaticPodType, "", resource.VersionUndefined))
	if err != nil {
		return err
	}

	// TODO: change this to proper teardown sequence

	for _, res := range list.Items {
		if err = r.Destroy(ctx, res.Metadata()); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *ControlPlaneStaticPodController) manageAPIServer(ctx context.Context, r controller.Runtime, logger *log.Logger, configResource *config.K8sControlPlane) error {
	cfg := configResource.APIServer()

	args := []string{
		"/go-runner",
		"/usr/local/bin/kube-apiserver",
		"--enable-admission-plugins=PodSecurityPolicy,NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeClaimResize,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,Priority,NodeRestriction", //nolint: lll
		"--advertise-address=$(POD_IP)",
		"--allow-privileged=true",
		fmt.Sprintf("--api-audiences=%s", cfg.ControlPlaneEndpoint),
		"--authorization-mode=Node,RBAC",
		"--bind-address=0.0.0.0",
		"--client-ca-file=/etc/kubernetes/secrets/ca.crt",
		"--requestheader-client-ca-file=/etc/kubernetes/secrets/front-proxy-ca.crt",
		"--requestheader-allowed-names=front-proxy-client",
		"--requestheader-extra-headers-prefix=X-Remote-Extra-",
		"--requestheader-group-headers=X-Remote-Group",
		"--requestheader-username-headers=X-Remote-User",
		"--proxy-client-cert-file=/etc/kubernetes/secrets/front-proxy-client.crt",
		"--proxy-client-key-file=/etc/kubernetes/secrets/front-proxy-client.key",
		"--cloud-provider=",
		"--enable-bootstrap-token-auth=true",
		"--tls-cipher-suites=TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_128_GCM_SHA256", //nolint: lll
		"--encryption-provider-config=/etc/kubernetes/secrets/encryptionconfig.yaml",
		"--audit-policy-file=/etc/kubernetes/secrets/auditpolicy.yaml",
		"--audit-log-path=-",
		"--audit-log-maxage=30",
		"--audit-log-maxbackup=3",
		"--audit-log-maxsize=50",
		"--profiling=false",
		"--etcd-cafile=/etc/kubernetes/secrets/etcd-client-ca.crt",
		"--etcd-certfile=/etc/kubernetes/secrets/etcd-client.crt",
		"--etcd-keyfile=/etc/kubernetes/secrets/etcd-client.key",
		fmt.Sprintf("--etcd-servers=%s", strings.Join(cfg.EtcdServers, ",")),
		"--insecure-port=0",
		"--kubelet-client-certificate=/etc/kubernetes/secrets/apiserver-kubelet-client.crt",
		"--kubelet-client-key=/etc/kubernetes/secrets/apiserver-kubelet-client.key",
		fmt.Sprintf("--secure-port=%d", cfg.LocalPort),
		fmt.Sprintf("--service-account-issuer=%s", cfg.ControlPlaneEndpoint),
		"--service-account-key-file=/etc/kubernetes/secrets/service-account.pub",
		"--service-account-signing-key-file=/etc/kubernetes/secrets/service-account.key",
		fmt.Sprintf("--service-cluster-ip-range=%s", cfg.ServiceCIDR),
		"--tls-cert-file=/etc/kubernetes/secrets/apiserver.crt",
		"--tls-private-key-file=/etc/kubernetes/secrets/apiserver.key",
		"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
	}

	for k, v := range cfg.ExtraArgs {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	return r.Update(ctx, k8s.NewStaticPod(k8s.ControlPlaneNamespaceName, "kube-apiserver", nil), func(r resource.Resource) error {
		r.(*k8s.StaticPod).SetPod(&v1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver-static",
				Namespace: "kube-system",
				Labels: map[string]string{
					"tier":    "control-plane",
					"k8s-app": "kube-apiserver",
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:    "kube-apiserver",
						Image:   cfg.Image,
						Command: args,
						Env: []v1.EnvVar{
							{
								Name: "POD_IP",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
						},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "ssl-certs",
								MountPath: "/etc/ssl/certs",
								ReadOnly:  true,
							},
							{
								Name:      "secrets",
								MountPath: "/etc/kubernetes/secrets",
								ReadOnly:  true,
							},
						},
					},
				},
				HostNetwork: true,
				SecurityContext: &v1.PodSecurityContext{
					RunAsNonRoot: pointer.ToBool(true),
					RunAsUser:    pointer.ToInt64(65534),
				},
				Volumes: []v1.Volume{
					{
						Name: "ssl-certs",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/etc/ssl/certs",
							},
						},
					},
					{
						Name: "secrets",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/etc/kubernetes/secrets",
							},
						},
					},
				},
			},
		})

		return nil
	})
}
