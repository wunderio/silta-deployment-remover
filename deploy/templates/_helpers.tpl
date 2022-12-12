{{- define "silta-cluster.ingress-api-version" }}
{{- if and ( ge $.Capabilities.KubeVersion.Major "1") ( ge $.Capabilities.KubeVersion.Minor "18" ) }}
networking.k8s.io/v1
{{- else }}
networking.k8s.io/v1beta1
{{- end }}
{{- end }}

{{- define "silta-cluster.cert-manager-api-version" }}
{{- if ( .Capabilities.APIVersions.Has "cert-manager.io/v1" ) }}
cert-manager.io/v1
{{- else }}
certmanager.k8s.io/v1alpha1
{{- end }}
{{- end }}
