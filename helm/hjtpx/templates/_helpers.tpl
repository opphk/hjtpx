{{- /*
Generate the NOTES.txt file
*/ -}}
{{- define "hjtpx.notes" -}}
{{- .Chart.Name }} v{{ .Chart.Version }}
{{- if .Values.ingress.enabled }}
Ingress URL: http://{{ (index .Values.ingress.hosts 0).host }}
{{- else }}
Service URL: http://{{ .Release.Name }}-hjtpx:{{ .Values.service.port }}
{{- end }}

{{- if .Values.monitoring.enabled }}
Prometheus metrics available at: /metrics
{{- end }}

{{- if .Values.postgresql.enabled }}
PostgreSQL is enabled and will be deployed.
{{- end }}

{{- if .Values.redis.enabled }}
Redis is enabled and will be deployed.
{{- end }}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "hjtpx.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "hjtpx.fullname" -}}
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
{{- define "hjtpx.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "hjtpx.labels" -}}
helm.sh/chart: {{ include "hjtpx.chart" . }}
{{ include "hjtpx.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "hjtpx.selectorLabels" -}}
app.kubernetes.io/name: {{ include "hjtpx.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "hjtpx.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "hjtpx.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
