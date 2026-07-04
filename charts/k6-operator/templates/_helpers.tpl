{{/*
Expand the name of the chart.
*/}}
{{- define "k6-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Create a default fully qualified app name, optionally appending a suffix.
The whole name is kept within the Kubernetes 63-char name limit (DNS naming
spec): the fullname part (not the suffix) is truncated, so the suffix is
always preserved. This keeps names that share a fullname but differ only by
suffix (e.g. the various Roles/RoleBindings) unique.
If release name contains chart name it will be used as a full name.
Usage: {{ include "k6-operator.fullname" (dict "context" $ "suffix" "-controller-manager") }}
       {{ include "k6-operator.fullname" (dict "context" $) }}
*/}}
{{- define "k6-operator.fullname" -}}
{{- $ctx := .context -}}
{{- $suffix := .suffix | default "" -}}
{{- $base := "" -}}
{{- if $ctx.Values.fullnameOverride -}}
{{- $base = $ctx.Values.fullnameOverride -}}
{{- else -}}
{{- $name := default $ctx.Chart.Name $ctx.Values.nameOverride -}}
{{- if contains $name $ctx.Release.Name -}}
{{- $base = $ctx.Release.Name -}}
{{- else -}}
{{- $base = printf "%s-%s" $ctx.Release.Name $name -}}
{{- end -}}
{{- end -}}
{{- printf "%s%s" ($base | trunc (int (sub 63 (len $suffix))) | trimSuffix "-") $suffix -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "k6-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "k6-operator.labels" -}}
helm.sh/chart: {{ include "k6-operator.chart" . }}
{{ include "k6-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: k6-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "k6-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k6-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "k6-operator.serviceAccountName" -}}
{{- if .Values.manager.serviceAccount.create }}
{{- default (include "k6-operator.fullname" (dict "context" .)) .Values.manager.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.manager.serviceAccount.name }}
{{- end }}
{{- end }}


{{- define "k6-operator.customLabels" -}}
  {{- if .Values.customLabels }}
    {{- with .Values.customLabels }}
      {{- toYaml . }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "k6-operator.podLabels" -}}
  {{- if .Values.podLabels }}
    {{- with .Values.podLabels }}
      {{- toYaml . }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "k6-operator.customAnnotations" -}}
  {{- if .Values.customAnnotations }}
    {{- with .Values.customAnnotations }}
      {{- toYaml . }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "k6-operator.podAnnotations" -}}
  {{- if .Values.podAnnotations }}
    {{- with .Values.podAnnotations }}
      {{- toYaml . }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "k6-operator.namespace" -}}
  {{- if eq .Release.Namespace "default" }}
    {{- printf "%v-system" .Release.Name | indent 1 }}
  {{- else }}
    {{- .Release.Namespace | indent 1 }}
  {{- end }}
{{- end -}}

{{/*
Create the --zap-devel flag for the manager
*/}}
{{- define "k6-operator.manager.zap-devel" -}}
{{- if hasKey .Values.manager.logging "development" }}{{ .Values.manager.logging.development | toString }}{{ else }}true{{ end }}
{{- end -}}

{{/*
Define env vars for the manager, taking into account whether deployment is namespaced.
*/}}
{{- define "k6-operator.manager.env" -}}
  {{- if or .Values.manager.env .Values.rbac.namespaced }}
    {{- printf "env:" | nindent 10 }}
  {{- end }}
  {{- if .Values.manager.env }}
    {{- with .Values.manager.env }}
      {{- toYaml . | nindent 12 }}
    {{- end }}
  {{- end }}
  {{- if .Values.rbac.namespaced }}
    {{- printf "- name: WATCH_NAMESPACE" | nindent 12 }}
    {{- printf "value: %s" (include "k6-operator.namespace" .) | nindent 14 }}
  {{- end }}
{{- end -}}