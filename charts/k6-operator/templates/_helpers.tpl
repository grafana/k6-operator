{{/*
Expand the name of the chart.
*/}}
{{- define "k6-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "k6-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

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
{{- default (include "k6-operator.fullname" .) .Values.manager.serviceAccount.name }}
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

{{- define "k6-operator.namespace" -}}
  {{- if eq .Release.Namespace "default" }}
    {{- printf "%v-system" .Release.Name | indent 1 }}
  {{- else }}
    {{- .Release.Namespace | indent 1 }}
  {{- end }}
{{- end -}}


{{- define "k6-operator.livenessProbe" -}}
  {{- if .Values.authProxy.livenessProbe }}
  livenessProbe:
    {{- toYaml .Values.authProxy.livenessProbe | nindent 12 }}
  {{- end }}
{{- end -}}

{{- define "k6-operator.readinessProbe" -}}
  {{- if .Values.authProxy.readinessProbe }}
  readinessProbe:
    {{- toYaml .Values.authProxy.readinessProbe | nindent 12 }}
  {{- end }}
{{- end -}}
