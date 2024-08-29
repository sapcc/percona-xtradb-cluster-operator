package statefulset

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	api "github.com/percona/percona-xtradb-cluster-operator/pkg/apis/pxc/v1"
	"github.com/percona/percona-xtradb-cluster-operator/pkg/pxc/app"
)

func EntrypointInitContainer(cr *api.PerconaXtraDBCluster, initImageName string, volumeName string) corev1.Container {
	initResources := cr.Spec.PXC.Resources
	if cr.Spec.InitContainer.Resources != nil {
		initResources = *cr.Spec.InitContainer.Resources
	}
	return corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: "/var/lib/mysql",
			},
		},
		Image:           initImageName,
		ImagePullPolicy: cr.Spec.PXC.ImagePullPolicy,
		Name:            "pxc-init",
		Command:         []string{"/pxc-init-entrypoint.sh"},
		SecurityContext: cr.Spec.PXC.ContainerSecurityContext,
		Resources:       initResources,
	}
}

func PermissionsInitContainer(cr *api.PerconaXtraDBCluster, initImageName string, volumeName string) corev1.Container {
	initResources := cr.Spec.PXC.Resources
	if cr.Spec.InitContainer.Resources != nil {
		initResources = *cr.Spec.InitContainer.Resources
	}
	// Default UID and GID if not specified
	var uid, gid int64 = 1001, 1001

	if cr.Spec.PXC.PodSecurityContext != nil {
		if cr.Spec.PXC.PodSecurityContext.RunAsUser != nil {
			uid = *cr.Spec.PXC.PodSecurityContext.RunAsUser
		}
		if cr.Spec.PXC.PodSecurityContext.RunAsGroup != nil {
			uid = *cr.Spec.PXC.PodSecurityContext.RunAsGroup
		}
	}

	return corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: "/var/lib/mysql",
			},
		},
		Image:           initImageName,
		ImagePullPolicy: cr.Spec.PXC.ImagePullPolicy,
		Name:            "pxc-permissions-init",
		Command: []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("chown -R %d:%d /var/lib/mysql", uid, gid),
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: pointer.Int64(0),
		},
		Resources: initResources,
	}
}

func PitrInitContainer(cluster *api.PerconaXtraDBCluster, initImageName string) corev1.Container {
	return corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      app.BinVolumeName,
				MountPath: app.BinVolumeMountPath,
			},
		},
		Image:           initImageName,
		ImagePullPolicy: cluster.Spec.Backup.ImagePullPolicy,
		Name:            "pitr-init",
		Command:         []string{"/pitr-init-entrypoint.sh"},
		SecurityContext: cluster.Spec.PXC.ContainerSecurityContext,
		Resources:       *cluster.Spec.InitContainer.Resources,
	}
}

func BackupInitContainer(cluster *api.PerconaXtraDBCluster, initImageName string, securityContext *corev1.SecurityContext) corev1.Container {
	return corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      app.BinVolumeName,
				MountPath: app.BinVolumeMountPath,
			},
		},
		Image:           initImageName,
		ImagePullPolicy: cluster.Spec.Backup.ImagePullPolicy,
		Name:            "backup-init",
		Command:         []string{"/backup-init-entrypoint.sh"},
		SecurityContext: securityContext,
		Resources:       *cluster.Spec.InitContainer.Resources,
	}
}

func HaproxyEntrypointInitContainer(cluster *api.PerconaXtraDBCluster, initImageName string) corev1.Container {
	return corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      app.BinVolumeName,
				MountPath: app.BinVolumeMountPath,
			},
		},
		Image:           initImageName,
		ImagePullPolicy: cluster.Spec.HAProxy.ImagePullPolicy,
		Name:            "haproxy-init",
		Command:         []string{"/haproxy-init-entrypoint.sh"},
		SecurityContext: cluster.Spec.HAProxy.ContainerSecurityContext,
		Resources:       *cluster.Spec.InitContainer.Resources,
	}
}

func ProxySQLEntrypointInitContainer(cluster *api.PerconaXtraDBCluster, initImageName string) corev1.Container {
	return corev1.Container{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      app.BinVolumeName,
				MountPath: app.BinVolumeMountPath,
			},
		},
		Image:           initImageName,
		ImagePullPolicy: cluster.Spec.ProxySQL.ImagePullPolicy,
		Name:            "proxysql-init",
		Command:         []string{"/proxysql-init-entrypoint.sh"},
		SecurityContext: cluster.Spec.ProxySQL.ContainerSecurityContext,
		Resources:       *cluster.Spec.InitContainer.Resources,
	}
}
