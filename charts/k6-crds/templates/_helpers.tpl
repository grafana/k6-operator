{{/*
Common labels
*/}}
{{- define "k6-crds.labels" -}}
helm.sh/chart: {{ include "k6-operator.chart" . }}
{{ include "k6-crds.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: k6-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "k6-crds.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k6-crds.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
