{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "zone-eu-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zone-eu-webhook.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "zone-eu-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "zone-eu-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "zone-eu-webhook.fullname" .) }}
{{- end -}}

{{- define "zone-eu-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "zone-eu-webhook.fullname" .) }}
{{- end -}}

{{- define "zone-eu-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "zone-eu-webhook.fullname" .) }}
{{- end -}}

{{- define "zone-eu-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "zone-eu-webhook.fullname" .) }}
{{- end -}}
