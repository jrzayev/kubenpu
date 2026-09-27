{
  _config:: {
    name: 'kubenpu',
    namespace: 'kubenpu',
    createNamespace: true,
    version: '0.1.1',

    image: {
      repository: 'ghcr.io/jrzayev/kubenpu',
      tag: '',
      pullPolicy: 'IfNotPresent',
    },
    imagePullSecrets: [],

    config: {
      debug: 0,
      intervalSeconds: 5,
      criTimeoutSeconds: 5,
      shutdownTimeoutSeconds: 5,
      cacheTTLSeconds: 60,
      queueSize: 4096,
      driPath: '/dev/dri',
      accelPath: '/dev/accel',
      sysfsDriPath: '/sys/class/drm',
      sysfsAccelPath: '/sys/class/accel',
      cgroupRootPath: '/sys/fs/cgroup',
      criSocketPath: '/run/containerd/containerd.sock',
    },

    mountAccel: false,

    metrics: { port: 8080 },

    service: {
      enabled: true,
      type: 'ClusterIP',
      port: 8080,
      annotations: {},
    },

    serviceMonitor: {
      enabled: false,
      interval: '30s',
      scrapeTimeout: '10s',
      labels: {},
      relabelings: [],
      metricRelabelings: [],
    },

    securityContext: {
      runAsUser: 0,
      runAsNonRoot: false,
      allowPrivilegeEscalation: false,
      readOnlyRootFilesystem: true,
      privileged: false,
      capabilities: {
        drop: ['ALL'],
        add: ['BPF', 'PERFMON'],
      },
    },

    resources: {
      requests: { cpu: '50m', memory: '64Mi' },
      limits: { memory: '256Mi' },
    },

    terminationGracePeriodSeconds: 30,

    livenessProbe: {
      initialDelaySeconds: 5,
      periodSeconds: 20,
      timeoutSeconds: 3,
      failureThreshold: 3,
    },

    readinessProbe: {
      initialDelaySeconds: 3,
      periodSeconds: 10,
      timeoutSeconds: 3,
      failureThreshold: 3,
    },

    updateStrategy: {
      type: 'RollingUpdate',
      rollingUpdate: { maxUnavailable: 1 },
    },

    priorityClassName: '',
    nodeSelector: {},
    tolerations: [{ operator: 'Exists' }],
    affinity: {},
    podAnnotations: {},
    podLabels: {},
  },

  local c = $._config,
  local criSocketDir = std.join('/', std.split(c.config.criSocketPath, '/')[:std.length(std.split(c.config.criSocketPath, '/')) - 1]),
  local imageTag = if c.image.tag != '' then c.image.tag else c.version,

  local selectorLabels = {
    'app.kubernetes.io/name': c.name,
    'app.kubernetes.io/instance': c.name,
  },

  local labels = selectorLabels {
    'app.kubernetes.io/version': c.version,
    'app.kubernetes.io/managed-by': 'tanka',
    'app.kubernetes.io/part-of': 'kubenpu',
  },

  local env = {
    KUBENPU_PORT: c.metrics.port,
    KUBENPU_DEBUG: c.config.debug,
    KUBENPU_INTERVAL: c.config.intervalSeconds,
    KUBENPU_CRI_TIMEOUT: c.config.criTimeoutSeconds,
    KUBENPU_SHUTDOWN_TIMEOUT: c.config.shutdownTimeoutSeconds,
    KUBENPU_CACHE_TTL: c.config.cacheTTLSeconds,
    KUBENPU_QUEUE_SIZE: c.config.queueSize,
    KUBENPU_DRI_PATH: c.config.driPath,
    KUBENPU_ACCEL_PATH: c.config.accelPath,
    KUBENPU_SYSFS_DRI_PATH: c.config.sysfsDriPath,
    KUBENPU_SYSFS_ACCEL_PATH: c.config.sysfsAccelPath,
    KUBENPU_CGROUP_ROOT_PATH: c.config.cgroupRootPath,
    KUBENPU_CRI_SOCKET_PATH: c.config.criSocketPath,
    KUBENPU_APP_VERSION: c.version,
  },

  local mount(name, path) = { name: name, mountPath: path, readOnly: true },
  local hostVolume(name, path) = { name: name, hostPath: { path: path, type: 'Directory' } },

  namespace: if c.createNamespace then {
    apiVersion: 'v1',
    kind: 'Namespace',
    metadata: { name: c.namespace },
  } else {},

  daemonset: {
    apiVersion: 'apps/v1',
    kind: 'DaemonSet',
    metadata: {
      name: c.name,
      namespace: c.namespace,
      labels: labels,
    },
    spec: {
      selector: { matchLabels: selectorLabels },
      updateStrategy: c.updateStrategy,
      template: {
        metadata: {
          labels: selectorLabels + c.podLabels,
          annotations: {
            'checksum/config': std.sha256(std.manifestJsonEx(c.config, '')),
          } + c.podAnnotations,
        },
        spec: {
          automountServiceAccountToken: false,
          terminationGracePeriodSeconds: c.terminationGracePeriodSeconds,
          hostPID: false,
          containers: [{
            name: 'agent',
            image: '%s:%s' % [c.image.repository, imageTag],
            imagePullPolicy: c.image.pullPolicy,
            securityContext: c.securityContext,
            ports: [{ name: 'metrics', containerPort: c.metrics.port, protocol: 'TCP' }],
            env: [{ name: k, value: std.toString(env[k]) } for k in std.objectFields(env)] + [{
              name: 'NODE_NAME',
              valueFrom: { fieldRef: { fieldPath: 'spec.nodeName' } },
            }],
            livenessProbe: { httpGet: { path: '/healthz', port: 'metrics' } } + c.livenessProbe,
            readinessProbe: { httpGet: { path: '/readyz', port: 'metrics' } } + c.readinessProbe,
            resources: c.resources,
            volumeMounts: [
              mount('cgroupfs', c.config.cgroupRootPath),
              mount('sysfs', '/sys'),
              mount('dri', c.config.driPath),
            ] + (if c.mountAccel then [mount('accel', c.config.accelPath)] else []) + [
              mount('cri-socket', criSocketDir),
            ],
          }],
          volumes: [
            hostVolume('cgroupfs', c.config.cgroupRootPath),
            hostVolume('sysfs', '/sys'),
            hostVolume('dri', c.config.driPath),
          ] + (if c.mountAccel then [hostVolume('accel', c.config.accelPath)] else []) + [
            hostVolume('cri-socket', criSocketDir),
          ],
        }
        + (if std.length(c.imagePullSecrets) > 0 then { imagePullSecrets: c.imagePullSecrets } else {})
        + (if c.priorityClassName != '' then { priorityClassName: c.priorityClassName } else {})
        + (if std.length(c.nodeSelector) > 0 then { nodeSelector: c.nodeSelector } else {})
        + (if std.length(c.tolerations) > 0 then { tolerations: c.tolerations } else {})
        + (if std.length(c.affinity) > 0 then { affinity: c.affinity } else {}),
      },
    },
  },

  service: if c.service.enabled then {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: c.name,
      namespace: c.namespace,
      labels: labels,
    } + (if std.length(c.service.annotations) > 0 then { annotations: c.service.annotations } else {}),
    spec: {
      type: c.service.type,
      clusterIP: 'None',
      selector: selectorLabels,
      ports: [{ name: 'metrics', port: c.service.port, targetPort: 'metrics', protocol: 'TCP' }],
    },
  } else {},

  serviceMonitor: if c.serviceMonitor.enabled then (
    assert c.service.enabled : 'serviceMonitor.enabled requires service.enabled';
    {
      apiVersion: 'monitoring.coreos.com/v1',
      kind: 'ServiceMonitor',
      metadata: {
        name: c.name,
        namespace: c.namespace,
        labels: labels + c.serviceMonitor.labels,
      },
      spec: {
        selector: { matchLabels: selectorLabels },
        namespaceSelector: { matchNames: [c.namespace] },
        endpoints: [
          {
            port: 'metrics',
            path: '/metrics',
            interval: c.serviceMonitor.interval,
            scrapeTimeout: c.serviceMonitor.scrapeTimeout,
          }
          + (if std.length(c.serviceMonitor.relabelings) > 0 then { relabelings: c.serviceMonitor.relabelings } else {})
          + (if std.length(c.serviceMonitor.metricRelabelings) > 0 then { metricRelabelings: c.serviceMonitor.metricRelabelings } else {}),
        ],
      },
    }
  ) else {},
}
