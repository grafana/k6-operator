{{- define "operator.labels" -}}
  {{- if .Values.customLabels }}
    {{- with .Values.customLabels }}
      {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- else}}
    {{ print "{}" }}
  {{- end }}
{{- end -}}

{{- define "operator.annotations" -}}
  {{- if .Values.customAnnotations }}
    {{- with .Values.customAnnotations }}
      {{- toYaml . }}
    {{- end }}
  {{- end }}
{{- end -}}

{{- define "operator.namespace" -}}
  {{- if eq .Release.Namespace "default" }}
    {{- printf "%v-system" .Release.Name | indent 1 }}
  {{- else }}
    {{- .Release.Namespace | indent 1 }}
  {{- end }}
{{- end -}}


{{- define "operator.livenessProbe" -}}
  {{- if .Values.authProxy.livenessProbe }}
  livenessProbe:
    {{- toYaml .Values.authProxy.livenessProbe | nindent 12 }}
  {{- end }}
{{- end -}}

{{- define "operator.readinessProbe" -}}
  {{- if .Values.authProxy.readinessProbe }}
  readinessProbe:
    {{- toYaml .Values.authProxy.readinessProbe | nindent 12 }}
  {{- end }}
{{- end -}}
