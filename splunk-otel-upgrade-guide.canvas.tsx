import {
  Card,
  CardHeader,
  CardBody,
  H1,
  H2,
  Text,
  Stack,
  Row,
  Grid,
  Callout,
  Pill,
  Table,
  Code,
  Divider,
  Spacer,
  Button,
  CollapsibleSection,
  useHostTheme,
  useCanvasState,
} from "cursor/canvas";

type Section =
  | "overview"
  | "renames"
  | "kafka"
  | "processors"
  | "splunk"
  | "receivers"
  | "exporters"
  | "ottl"
  | "checklist";

const NAV: { id: Section; label: string }[] = [
  { id: "overview", label: "Overview" },
  { id: "renames", label: "Component Renames" },
  { id: "kafka", label: "Kafka Migration" },
  { id: "processors", label: "Processors" },
  { id: "receivers", label: "Receivers" },
  { id: "exporters", label: "Exporters" },
  { id: "ottl", label: "OTTL Changes" },
  { id: "splunk", label: "Splunk-Specific" },
  { id: "checklist", label: "Upgrade Checklist" },
];

const RENAMES: [string, string, string, string][] = [
  ["receiver", "filelog", "file_log", "0.149"],
  ["receiver", "fluentforward", "fluent_forward", "0.151"],
  ["receiver", "hostmetrics", "host_metrics", "0.151"],
  ["receiver", "k8sobjects", "k8s_objects", "0.151"],
  ["receiver", "kubeletstats", "kubelet_stats", "0.152"],
  ["receiver", "kafkametrics", "kafka_metrics", "0.152"],
  ["receiver", "httpcheck", "http_check", "0.150"],
  ["receiver", "tcpcheck", "tcp_check", "0.152"],
  ["receiver", "tcplog", "tcp_log", "0.150"],
  ["receiver", "udplog", "udp_log", "0.150"],
  ["receiver", "windowseventlog", "windows_event_log", "0.150"],
  ["receiver", "sshcheck", "ssh_check", "0.151"],
  ["receiver", "tlscheck", "tls_check", "0.150"],
  ["receiver", "filestats", "file_stats", "0.151"],
  ["receiver", "namedpipe", "named_pipe", "0.150"],
  ["receiver", "cloudfoundry", "cloud_foundry", "0.152"],
  ["receiver", "azureeventhub", "azure_event_hub", "0.145"],
  ["receiver", "mongodbatlas", "mongodb_atlas", "0.145"],
  ["processor", "k8sattributes", "k8s_attributes", "0.146"],
  ["processor", "metricstransform", "metrics_transform", "0.152"],
  ["processor", "logdedup", "log_dedup", "0.151"],
  ["processor", "resourcedetection", "resource_detection", "0.153"],
  ["connector", "spanmetrics", "span_metrics", "0.151"],
  ["connector", "servicegraph", "service_graph", "0.151"],
  ["exporter", "loadbalancing", "load_balancing", "0.153"],
];

const REMOVED_MONITORS: [string, string, string][] = [
  ["sapm exporter", "otlphttp exporter", "0.147"],
  ["sapm receiver", "otlp receiver", "0.135"],
  ["routingprocessor", "routing connector", "0.134"],
  ["collectd/apache", "Apache receiver", "0.149"],
  ["collectd/cpufreq", "hostmetrics receiver", "0.149"],
  ["collectd/memory", "hostmetrics receiver (memory scraper)", "0.149"],
  ["collectd/opcache", "PHP SDK for OpenTelemetry", "0.149"],
  ["collectd/php-fpm", "PHP SDK for OpenTelemetry", "0.149"],
  ["collectd/processes", "hostmetrics receiver (process scraper)", "0.149"],
  ["collectd/systemd", "systemd receiver", "0.149"],
  ["collectd/uptime", "hostmetrics receiver (system scraper)", "0.149"],
  ["collectd/zookeeper", "zookeeper receiver", "0.149"],
  ["smartagent/ntp", "NTP receiver", "0.149"],
  ["smartagent/postgresql", "postgresql receiver", "0.149"],
  ["collectd/protocols", "hostmetrics receiver (network scraper)", "0.149"],
  ["cadvisor", "prometheus receiver", "0.144"],
  ["collectd/chrony", "chrony receiver", "0.144"],
  ["collectd/cpu", "cpu monitor", "0.144"],
  ["collectd/couchbase", "prometheus receiver", "0.144"],
  ["haproxy plugin", "haproxy receiver", "0.144"],
  ["heroku plugin", "resourcedetection processor (heroku)", "0.144"],
  ["kubelet-stats / kubelet-metrics", "kubeletstats receiver", "0.144"],
  ["mongodbatlas monitor", "mongodbatlas receiver", "0.144"],
  ["nagios", "No replacement", "0.144"],
  ["collectd/nginx", "nginx receiver", "0.144"],
  ["collectd/rabbitmq", "RabbitMQ receiver", "0.144"],
  ["collectd/redis", "Redis receiver", "0.144"],
  ["windows-legacy", "hostmetrics + windowsperfcounters receivers", "0.144"],
  ["collectd/spark", "Apache Spark receiver", "0.144"],
  ["collectd/activemq", "jmxreceiver (target: activemq)", "0.131"],
  ["collectd/cassandra", "jmxreceiver (target: cassandra)", "0.131"],
  ["collectd/hadoop", "jmxreceiver (target: hadoop)", "0.131"],
  ["collectd/kafka (collectd)", "jmxreceiver (target: kafka)", "0.131"],
  ["collectd/kafka-consumer", "jmxreceiver (target: kafka)", "0.131"],
  ["collectd/kafka-producer", "jmxreceiver (target: kafka)", "0.131"],
  ["collectd/solr", "jmxreceiver (target: solr)", "0.131"],
  ["collectd/tomcat", "jmxreceiver (target: tomcat)", "0.131"],
  ["smartagent/jaeger-grpc", "jaeger receiver (grpc protocol)", "0.131"],
  ["collectd/jenkins", "Jenkins OpenTelemetry plugin", "0.144"],
  ["FluentD (all installers)", "filelog receiver", "0.144–0.145"],
  ["ecs_task_observer extension", "None (removed upstream)", "0.140"],
  ["migratecheckpoint command", "None (FluentD sidecar gone)", "0.139"],
];

type ChecklistGroup = { group: string; items: string[] };
const CHECKLIST: ChecklistGroup[] = [
  {
    group: "Before Upgrading — Review",
    items: [
      "Back up your current collector configuration files",
      "Review all component names in your config against the Renames table",
      "Audit any FluentD / SmartAgent monitors in use — find replacements before upgrading",
      "Check if you use kafka exporter or kafkametrics receiver (client_id default changed sarama→otel-collector in 0.123 — verify Kafka ACLs)",
      "Check if you use kafka receiver (client_id config was broken before 0.130 — default changes to otel-collector in 0.130 — verify Kafka ACLs)",
      "Check Kafka components for auth.plain_text — deprecated in 0.123, migrate to auth.sasl with mechanism: PLAIN",
      "Search config for 'sapm' exporter or receiver usage — migrate to otlphttp",
      "Check if routingprocessor is used — migrate to routing connector (remove match_once: it was removed in 0.120)",
      "Verify any feature gates you have set explicitly — many have been promoted or removed",
      "Check service::telemetry::address — must migrate to readers: format under metrics: (gate stable in 0.128, fallback flag FAILS startup)",
      "Check signalfx exporter for translation_rules — must migrate to transform processor",
      "Check awss3 exporter for s3_partition — replace with s3_partition_format",
      "Check prometheusremotewrite exporter for export_created_metric — remove if present",
      "Check all exporters for sending_queue::blocking — replace with block_on_overflow (removed 0.129)",
      "Check transform processors for mixed Basic/Advanced config style — must use one or the other",
      "Check k8sattributes node_from_env_var — referenced env var must always be set or remove the field",
      "Check splunkenterprise receiver — all metrics now opt-in (except splunk.health); re-enable as needed",
      "Update dashboards referencing prometheus resource attributes: net.host.name→server.address, net.host.port→server.port, http.scheme→url.scheme (changed 0.126, permanent from 0.129)",
      "Update dashboards referencing internal metric names with dots vs underscores (Prometheus 3.0 change, 0.120)",
      "Remove deprecated feature gates: normalizeProcessCPUUtilization, k8sattr.rfc3339 (removed 0.120/0.123)",
    ],
  },
  {
    group: "Configuration File Updates",
    items: [
      "Rename all legacy component names to snake_case (see Renames table)",
      "Remove 'default_fetch_size' from kafka receiver config (Sarama-only option, removed 0.144)",
      "Remove top-level 'topic' and 'encoding' from kafka exporter — use per-signal fields (removed 0.148)",
      "Remove 'report_extra_scrape_metrics' from prometheus receiver config (removed 0.149)",
      "Remove 'use_start_time_metric' and 'start_time_metric_regex' from prometheus receiver (removed 0.143)",
      "Remove 'attributes' field from resourcedetection processor config (removed 0.142)",
      "Remove min_size_items/max_size_items from sending_queue::batch config — use min_size/max_size (removed 0.123)",
      "Remove export_created_metric from prometheusremotewrite exporter (removed 0.123)",
      "Remove translation_rules from signalfx exporter — migrate to transform processor (removed 0.121)",
      "Remove s3_partition from awss3 exporter — use s3_partition_format strftime string (removed 0.121)",
      "Replace null values with 0 in HTTP client max_idle_conns / idle_conn_timeout fields (0.121+)",
      "If using cumulativetodelta processor, add explicit max_staleness: 0 if you need infinite retention",
      "If using splunk_hec exporter with 'batcher' key, migrate to sending_queue::batch (removed 0.151)",
      "If using windowseventlog and need old array format: set event_data_format: array (changed 0.148)",
      "If using mysql receiver: re-enable query_sample if needed (off by default since 0.148)",
      "If using postgresql receiver: re-enable top_query/query_sample if needed (off by default since 0.148)",
      "If using kubeletstats: check if deprecated attributes (aws.volume.id, etc.) are needed (disabled 0.150)",
      "Remove 'faas.id' references — replaced with 'faas.instance' (resourcedetection, 0.147)",
      "Azure resource detection: update 'azure_eks' to 'azure.eks' and 'azure_vm' to 'azure.vm' (0.147)",
      "Update dashboard/alert thresholds for kubeletstats CPU utilization → usage metric names (0.125+)",
      "Update dashboard/alert thresholds for sqlserver db.lock_timeout — unit changed ms → seconds (0.124)",
      "Update OTTL/dashboards: sqlserver computer_name/instance_name are now resource attributes (0.125)",
      "Update OTTL/dashboards: otelcol.component.kind values now lowercase (0.125)",
      "Update OTTL/dashboards: activedirectoryds distingushed_names typo fixed → distinguished_names (0.120)",
    ],
  },
  {
    group: "Network / Infrastructure",
    items: [
      "Update firewall/allowlists: SignalFx exporter default endpoints changed to *.observability.splunkcloud.com (0.151)",
      "Update Windows MSI download URL from dl.signalfx.com to dl.observability.splunkcloud.com (0.151)",
      "Verify Prometheus remote write endpoint is Prometheus 3.8.0+ — RW 2.0 rc.4 incompatible with older (0.142)",
    ],
  },
  {
    group: "Metrics & Dashboards",
    items: [
      "Update dashboard queries filtering on service_name/service_instance_id/service_version on individual metrics — now only in target_info (0.149)",
      "Update httpcheck timing metrics dashboards — now in nanoseconds not milliseconds (0.153)",
      "Update ECS memory metric unit filters using 'Megabytes' string — now 'MiB' (0.148)",
      "spanmetrics connector: 'collector.instance.id' attribute now added to all metrics — check cardinality impact (0.152)",
      "OTel internal metric units changed to singular form: {request} not {requests} (0.148)",
      "MongoDB receiver: 'database' is no longer a resource attribute — now a metric-level 'db.namespace' (0.147)",
    ],
  },
  {
    group: "Processors — Behavior Changes",
    items: [
      "filter processor: default error_mode is now 'ignore' (was 'propagate') — add error_mode: propagate to restore (0.153)",
      "transform processor: default error_mode is now 'ignore' (was 'propagate') — add error_mode: propagate to restore (0.153)",
      "OTTL statements that were silently failing due to type mismatches will now surface errors — review all OTTL configs (0.150+)",
      "tail_sampling: invert decisions are permanently disabled — migrate to drop policies (0.144/0.152)",
      "truncate_all OTTL function: now UTF-8 safe by default — set utf8_safe: false to restore old byte-level behavior (0.148)",
    ],
  },
  {
    group: "Feature Gate Removals",
    items: [
      "Remove --feature-gates=exporter.kafkaexporter.UseFranzGo (Franz-go is now mandatory, gate removed 0.144)",
      "Remove --feature-gates=receiver.kafkareceiver.UseFranzGo (Franz-go is now mandatory, gate removed 0.144)",
      "Remove --feature-gates=processor.tailsamplingprocessor.disableinvertdecisions (gate stabilized 0.152)",
      "Remove --feature-gates=receiver.jaeger.DisableRemoteSampling (gate removed 0.153)",
      "Remove --feature-gates=pkg.translator.prometheus.NormalizeName (gate removed 0.151)",
      "Remove --feature-gates=processor.resourcedetection.removeGCPFaasID (gate removed 0.147)",
      "Remove --feature-gates=processor.resourcedetection.propagateerrors (gate removed 0.147, now always on)",
      "Remove --feature-gates=processor.transform.ConvertBetweenSumAndGaugeMetricContext (gate removed 0.150)",
      "Remove --feature-gates=telemetry.disableHighCardinalityMetrics (gate removed 0.144)",
      "Remove --feature-gates=service.noopTracerProvider (gate removed 0.144)",
      "Remove --feature-gates=+clickhouse.json (gate removed 0.153, use json: true in config instead)",
    ],
  },
  {
    group: "Post-Upgrade Verification",
    items: [
      "Run: otelcol validate --config=/etc/otelcol/config.yaml before deploying",
      "Verify all pipelines start without errors in the collector logs",
      "Confirm metrics are flowing to Splunk Observability / Splunk HEC",
      "Validate that component rename aliases are working (deprecation warnings expected — this is OK)",
      "Check OTTL transform/filter pipelines for newly surfaced errors from type mismatches",
      "Monitor for 'collector.instance.id' attribute appearing in spanmetrics — update any cardinality limits",
      "Test any Kafka integrations end-to-end with the new Franz-go-only config",
    ],
  },
];

export default function UpgradeGuide() {
  const theme = useHostTheme();
  const [active, setActive] = useCanvasState<Section>("section", "overview");

  return (
    <Stack gap={0} style={{ minHeight: "100vh", background: theme.bg.editor }}>
      {/* ── Header ── */}
      <Stack
        gap={12}
        style={{
          padding: "24px 32px 20px",
          borderBottom: `1px solid ${theme.stroke.tertiary}`,
          background: theme.bg.chrome,
        }}
      >
        <Row gap={12} align="center">
          <H1 style={{ margin: 0 }}>Splunk OTel Collector Upgrade Guide</H1>
          <Pill tone="info" active>v0.120.0 → v0.153.0</Pill>
        </Row>
        <Text tone="secondary">
          Comprehensive reference for all breaking changes, component renames, removed monitors, and
          required configuration updates across 33 releases.
        </Text>
        <Row gap={8} wrap>
          {NAV.map((n) => (
            <Button
              key={n.id}
              variant={active === n.id ? "primary" : "secondary"}
              onClick={() => setActive(n.id)}
            >
              {n.label}
            </Button>
          ))}
        </Row>
      </Stack>

      {/* ── Body ── */}
      <Stack gap={24} style={{ padding: "28px 32px", maxWidth: 1100 }}>

        {/* ────────── OVERVIEW ────────── */}
        {active === "overview" && (
          <Stack gap={20}>
            <H2>Upgrade Overview</H2>
            <Callout tone="warning" title="Large version jump">
              This spans 33 releases with many independent breaking changes. Validate in a staging
              environment and work through the Upgrade Checklist section before touching production.
            </Callout>
            <Grid columns={3} gap={16}>
              <Card>
                <CardHeader>Component Renames</CardHeader>
                <CardBody>
                  <Text>
                    The largest change is a systematic rename of 30+ components to{" "}
                    <Code>snake_case</Code>. All old names have deprecated aliases — they still work
                    today but will be removed. Update your config before aliases are dropped.
                  </Text>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Kafka: Sarama Removed</CardHeader>
                <CardBody>
                  <Text>
                    The legacy Sarama client is fully removed from both the Kafka receiver and
                    exporter. Franz-go is now mandatory. Remove <Code>default_fetch_size</Code>,{" "}
                    <Code>resolve_canonical_bootstrap_servers_only</Code>, and the top-level{" "}
                    <Code>topic</Code>/<Code>encoding</Code> exporter fields.
                  </Text>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>FluentD + Smart Agent</CardHeader>
                <CardBody>
                  <Text>
                    FluentD is fully removed from all Splunk installers. 30+ Smart Agent collectd
                    monitors are permanently removed. Every removed monitor has a native OTel
                    receiver replacement — see the Splunk-Specific section.
                  </Text>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Processor Error Mode Change</CardHeader>
                <CardBody>
                  <Text>
                    <Code>filter</Code> and <Code>transform</Code> processors now default to{" "}
                    <Code>error_mode: ignore</Code> (was <Code>propagate</Code>). Errors that would
                    previously fail pipeline batches now silently pass. Review your OTTL configs.
                  </Text>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>SignalFx URL Change</CardHeader>
                <CardBody>
                  <Text>
                    Default <Code>api_url</Code> and <Code>ingest_url</Code> derived from{" "}
                    <Code>realm</Code> now point to <Code>*.observability.splunkcloud.com</Code>{" "}
                    instead of <Code>*.signalfx.com</Code>. Update firewall allowlists if needed.
                  </Text>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Internal Metrics Labels Removed</CardHeader>
                <CardBody>
                  <Text>
                    <Code>service_name</Code>, <Code>service_instance_id</Code>, and{" "}
                    <Code>service_version</Code> are no longer stamped on every individual internal
                    metric series — they now only appear in <Code>target_info</Code>. Update
                    dashboard queries that join on these labels.
                  </Text>
                </CardBody>
              </Card>
            </Grid>

            <Row gap={10} wrap align="center" style={{ padding: "2px 0 4px" }}>
              <Text tone="secondary" style={{ fontSize: 12, fontWeight: 500 }}>Card impact level:</Text>
              <Pill tone="success" active>P3 Advisory</Pill>
              <Text tone="secondary" style={{ fontSize: 12 }}>no config change needed</Text>
              <Text tone="tertiary" style={{ fontSize: 12 }}>·</Text>
              <Pill tone="warning" active>P2 Degrading</Pill>
              <Text tone="secondary" style={{ fontSize: 12 }}>config or planning required</Text>
              <Text tone="tertiary" style={{ fontSize: 12 }}>·</Text>
              <Pill tone="danger" active>P1 Breaking</Pill>
              <Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <H2>Breaking Changes by Release</H2>
            <Table
              headers={["Version", "Key Breaking Changes"]}
              rows={[
                ["0.120", "prometheus receiver: Prometheus 3.0 scraper — dots no longer escaped in metric names; internal metric renames (processor_filter_* → processor_filter.*); resource attrs service_name/instance_id/version → service.name/instance.id/version; routing connector match_once removed; tail_sampling decision timer us→ms; activedirectoryds distingushed_names typo fixed"],
                ["0.121", "signalfx exporter: translation_rules removed; awss3 exporter: s3_partition → s3_partition_format; confighttp: max_idle_conns/etc type null→integer; k8sattr.fieldExtractConfigRegex.disallow gate to stable; Oracle Linux 7 support dropped"],
                ["0.122", "batch processor telemetry no longer at 'basic' level — switch to 'normal'; sqlserver receiver: X.509 certs must have positive serial number"],
                ["0.123", "service::telemetry::address silently ignored — use readers: format in metrics: (gate stable in 0.128, fallback flag FAILS startup from 0.128+); min_size_items/max_size_items removed from batch config; prometheusremotewrite: export_created_metric removed; kafka EXPORTER + kafkametrics client_id default changed sarama→otel-collector (ACL breakage!); auth.plain_text deprecated; k8sattr.rfc3339 gate removed"],
                ["0.124", "transform processor: Basic Config and Advanced Config cannot mix; splunkenterprise receiver: all metrics now opt-in except splunk.health; sqlserver db.lock_timeout unit ms→seconds; kafka auth::tls deprecated in favor of top-level tls:"],
                ["0.125", "otelcol.component.kind attribute values now lowercase; telemetry.newPipelineTelemetry on by default for logs/traces (scope attrs); kubeletstats enableCPUUsageMetrics gate to beta (cpu.utilization → cpu.usage; gate stable/permanent in 0.130); k8sattributes node_from_env_var is now a hard startup error if env var unset; sqlserver host.name/computer.name/instance.name moved to resource attributes"],
                ["0.126", "sqlserver receiver: event names changed; prometheus receiver: RemoveLegacyResourceAttributes gate beta — net.host.name→server.address, net.host.port→server.port, http.scheme→url.scheme (dashboards break!)"],
                ["0.127", "New Pipeline Component Telemetry size metrics enabled by default (billing impact); sqlserver page.life_expectancy now emitted for all configs (not just Windows)"],
                ["0.128", "sqlserver event flag renames (top_query_collection.enabled → events.\"db.server.top_query\".enabled); OTTL profile lookup tables removed from profile object"],
                ["0.129", "sending_queue::blocking field removed — use block_on_overflow (startup failure); prometheus RemoveLegacyResourceAttributes gate promoted to STABLE (net.host.name→server.address etc. now permanent); kafkareceiver internal metrics restructured"],
                ["0.130", "Kafka RECEIVER (Sarama) client ID config now honoured — default changes sarama→otel-collector (breaks ACLs); OTLP exporter deprecated batcher config removed (use queuebatch); kubeletstats enableCPUUsageMetrics gate stable"],
                ["0.131", "8 collectd/genericjmx monitors removed (activemq, cassandra, hadoop, kafka, kafka-consumer, kafka-producer, solr, tomcat); smartagent/jaeger-grpc removed; IIS app pool metrics enabled by default"],
                ["0.132", "postgresql: query_sample_collection.enabled + top_query_collection.enabled flags removed — both now enabled by default"],
                ["0.134", "routingprocessor removed — use routing connector; normalize_gcp converter removed"],
                ["0.135", "sapm receiver removed; access_token_passthrough DEPRECATED in signalfx receiver; k8sattributes allowLabelsAnnotationsSingular gate introduced (alpha)"],
                ["0.136", "kubeletstats: no-op config sections for disabled CPU utilization metrics now cause startup failure; Kafka client metrics publishing disabled"],
                ["0.137", "access_token_passthrough REMOVED from signalfx receiver; spanmetrics excludeResourceMetrics gate added (opt-in, becomes default in 0.138)"],
                ["0.138", "SignalFx receiver removed from all default configs except gateway; sending_queue::batch has non-trivial defaults when set to empty"],
                ["0.139", "migratecheckpoint command removed; ciscoos receiver renamed from ciscoosreceiver; sqlserver lookback_time requires 's' suffix"],
                ["0.140", "ecs_task_observer removed; SignalFx receiver removed from gateway config; prometheus receiver no longer adjusts start time by default; tail_sampling latency metric replaced with cpu time metric"],
                ["0.141", "Kafka Franz-go to Stable for receiver+exporter (Sarama removed in 0.144); kafka receiver top-level topic/encoding removed; docker_observer Docker API default 1.24→1.44"],
                ["0.142", "Config debug endpoint (localhost:55554) removed; PRW 2.0 rc.4 requires Prometheus 3.8+; cumulativetodelta max_staleness default changed; resourcedetection attributes field removed; dockerstats Docker API default 1.24→1.44"],
                ["0.143", "prometheus: use_start_time_metric + start_time_metric_regex removed"],
                ["0.144", "Kafka Sarama fully removed; 20+ Smart Agent monitors removed incl. collectd/jenkins; FluentD removed from installers; tail_sampling invert decisions off by default"],
                ["0.145", "FluentD removed from RPM/DEB/PowerShell; faas.id → faas.instance; azure_event_hub UseAzeventhubs gate stable; resourcedetection removeGCPFaasID gate stable"],
                ["0.146", "k8sattributes semantic convention feature gates added (alpha): EmitV1K8sConventions + DontEmitV0K8sConventions; signalfx receiver formally deprecated (use OTLP receiver instead)"],
                ["0.147", "resourcedetection: removeGCPFaasID gate removed; Azure EKS/VM cloud platform values changed (azure_eks → azure.eks); kafka receiver old topic/exclude_topic fields removed; MongoDB database attribute moved to db.namespace; sapm exporter removed"],
                ["0.148", "Kafka exporter top-level topic/encoding removed; kafka exporter batching requires explicit metadata_keys; splunk_hec batcher removed; Windows event_data flat map + rendering_info/user_data keys added; ECS memory units MiB; OTel metric units singular; MySQL/PostgreSQL query defaults off"],
                ["0.149", "7 collectd monitors + smartagent/ntp + smartagent/postgresql removed; includeconfigsource delete_files removed; prometheus report_extra_scrape_metrics removed; service_name/id/version labels removed from individual metrics"],
                ["0.150", "OTTL setters now return errors for type mismatches; kubeletstats deprecated attributes disabled by default"],
                ["0.151", "SignalFx URLs → *.observability.splunkcloud.com; Windows MSI URL changed; default configs updated to canonical names; Jaeger DisableRemoteSampling gate stable; prometheus feature gates removed; k8s_attributes otelcol.k8s.pod.association metric disabled"],
                ["0.152", "spanmetrics includeCollectorInstanceID gate to beta (adds collector.instance.id); tail_sampling disableinvertdecisions stabilized; kafka_metrics Sarama removed; prometheusremotewrite add_metric_suffixes deprecated"],
                ["0.153", "filter/transform default error_mode=ignore; OTTL datapoint setters type-strict; httpcheck metrics now in nanoseconds; Jaeger DisableRemoteSampling gate removed; clickhouse.json gate removed; Kafka resolve_canonical_bootstrap_servers_only + auth.sasl.version deprecated; signalfx receiver REMOVED from Splunk distribution"],
              ]}
              striped
              stickyHeader
            />
          </Stack>
        )}

        {/* ────────── COMPONENT RENAMES ────────── */}
        {active === "renames" && (
          <Stack gap={20}>
            <H2>Component Renames — snake_case Migration</H2>
            <Callout tone="info" title="Deprecated aliases still work today">
              All legacy names remain available as deprecated aliases in 0.153. They will be removed
              in a future release. Migrate your config now to avoid startup failures later.
            </Callout>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill>
              <Text tone="secondary" style={{ fontSize: 12 }}>All renames in this section carry <strong>P3 Advisory</strong> impact — aliases still work in 0.153. No config change causes an immediate failure, but migration is required before aliases are removed.</Text>
            </Row>
            <Text>
              A global initiative (#45339) renamed ~30 components to <Code>snake_case</Code> between
              versions 0.145 and 0.153. Update every receiver, processor, connector, and exporter key
              in your <Code>config.yaml</Code>.
            </Text>
            <Table
              headers={["Type", "Old Name (Deprecated)", "New Canonical Name", "Deprecated Since"]}
              rows={RENAMES}
              striped
              stickyHeader
            />
            <Card>
              <CardHeader>Migration Example</CardHeader>
              <CardBody>
                <Stack gap={12}>
                  <Text weight="semibold">Before — using old names:</Text>
                  <Code>{`receivers:
  hostmetrics:
    collection_interval: 10s
  filelog:
    include: [/var/log/*.log]
  k8sobjects: {}

processors:
  k8sattributes: {}
  resourcedetection:
    detectors: [gcp, ec2]

connectors:
  spanmetrics: {}

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [k8sattributes]`}</Code>
                  <Text weight="semibold" style={{ marginTop: 8 }}>After — canonical snake_case names:</Text>
                  <Code>{`receivers:
  host_metrics:
    collection_interval: 10s
  file_log:
    include: [/var/log/*.log]
  k8s_objects: {}

processors:
  k8s_attributes: {}
  resource_detection:
    detectors: [gcp, ec2]

connectors:
  span_metrics: {}

service:
  pipelines:
    metrics:
      receivers: [host_metrics]
      processors: [k8s_attributes]`}</Code>
                </Stack>
              </CardBody>
            </Card>
          </Stack>
        )}

        {/* ────────── KAFKA ────────── */}
        {active === "kafka" && (
          <Stack gap={20}>
            <H2>Kafka Migration — Sarama → Franz-go</H2>
            <Callout tone="danger" title="Sarama permanently removed in 0.144">
              The legacy Sarama Kafka client is permanently removed from both the Kafka receiver and
              exporter. Franz-go is the only supported client. All Sarama-only configuration keys
              must be removed or the collector will fail to start.
            </Callout>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill><Text tone="secondary" style={{ fontSize: 12 }}>no config change  ·  </Text>
              <Pill tone="warning" active>P2 Degrading</Pill><Text tone="secondary" style={{ fontSize: 12 }}>config/planning required  ·  </Text>
              <Pill tone="danger" active>P1 Breaking</Pill><Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <Grid columns={2} gap={16}>
              <Card>
                <CardHeader>Kafka Receiver — Keys to Remove <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text weight="semibold">Remove these keys (0.144–0.147):</Text>
                    <Code>{`# REMOVED in 0.144 (Sarama-only):
default_fetch_size: 1048576

# REMOVED in 0.141/0.147:
topic: my-topic           # use topics: [my-topic]
exclude_topic: bad-topic  # use exclude_topics: [bad-topic]

# Feature gate no longer needed:
# --feature-gates=receiver.kafkareceiver.UseFranzGo`}</Code>
                    <Text weight="semibold" style={{ marginTop: 8 }}>Correct form:</Text>
                    <Code>{`receivers:
  kafka:
    topics: [my-topic, other-topic]
    exclude_topics: [excluded-topic]
    # Franz-go equivalent for fetch size:
    max_fetch_size: 50MiB`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Kafka Exporter — Keys to Remove <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text weight="semibold">Remove top-level fields (removed 0.148):</Text>
                    <Code>{`exporters:
  kafka:
    topic: my-topic      # REMOVED (deprecated since 0.124)
    encoding: otlp_proto # REMOVED (deprecated since 0.124)`}</Code>
                    <Text weight="semibold" style={{ marginTop: 8 }}>Use per-signal fields instead:</Text>
                    <Code>{`exporters:
  kafka:
    logs:
      topic: my-logs-topic
      encoding: otlp_proto
    metrics:
      topic: my-metrics-topic
      encoding: otlp_proto
    traces:
      topic: my-traces-topic
      encoding: otlp_proto`}</Code>
                    <Text weight="semibold" style={{ marginTop: 8 }}>Also remove if set (deprecated, no-ops):</Text>
                    <Code>{`auth:
  sasl:
    version: 1  # franz-go auto-negotiates, no effect

# Also no-op (no franz-go equivalent):
resolve_canonical_bootstrap_servers_only: true`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Kafka Receiver (Sarama) — Client ID Now Honoured (0.130) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      <strong>Silent data loss risk.</strong> In the Sarama-based kafka RECEIVER,
                      the <Code>client_id</Code> config was broken before 0.130 — always set to
                      <Code>"sarama"</Code> regardless of config. In 0.130 the config is honoured
                      and the default is now <Code>"otel-collector"</Code>. Kafka ACLs based on
                      client ID <Code>"sarama"</Code> will silently break. See also: P2-34 for
                      kafka EXPORTER and kafkametrics changes in 0.123.
                    </Callout>
                    <Code>{`# OPTION A — Pin the old client ID (no Kafka ACL changes needed):
receivers:
  kafka:
    client_id: sarama   # preserves legacy ACL match

exporters:
  kafka:
    client_id: sarama   # preserves legacy ACL match

# OPTION B — Update Kafka ACLs to allow "otel-collector":
# kafka-acls.sh --bootstrap-server <host>:9092 \
#   --add --allow-principal User:<collector-user> \
#   --operation Write --topic <topic> \
#   --client-id otel-collector
# Then remove old "sarama" ACL entry.
# Also update dashboards/alerts filtering on client.id.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Kafka Exporter — Batching Partitioner metadata_keys (0.148) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      If you have batching enabled with metadata partitioning, you must now explicitly
                      set <Code>metadata_keys</Code>. The automatic wiring was removed in 0.148.
                    </Callout>
                    <Code>{`# Before 0.148 (implicit wiring):
exporters:
  kafka:
    sending_queue:
      batch:
        enabled: true
    include_metadata_keys: [tenant_id]

# After 0.148 (explicit metadata_keys required):
exporters:
  kafka:
    sending_queue:
      batch:
        enabled: true
        partition:
          metadata_keys: [tenant_id]   # must match include_metadata_keys
    include_metadata_keys: [tenant_id]`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kafka_metrics Receiver <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={8}>
                    <Text>
                      In 0.152, <Code>receiver.kafkametricsreceiver.UseFranzGo</Code> was promoted to
                      stable and the Sarama implementation was removed entirely. The gate will be
                      removed in 0.154.
                    </Text>
                    <Code>{`# Remove this flag from startup if present:
# --feature-gates=+receiver.kafkametricsreceiver.UseFranzGo

# Rename the receiver (alias still available in 0.153):
# kafkametrics  →  kafka_metrics`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Kafka Deprecated No-Op Fields (0.153) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      Two Kafka client config fields are now deprecated because they became no-ops
                      after the Sarama → Franz-go migration. They still parse without error but have
                      no effect and will be removed in a future release.
                    </Text>
                    <Code>{`# DEPRECATED in 0.153 (no-ops — remove to avoid future failure):
kafka:
  resolve_canonical_bootstrap_servers_only: true
    # Franz-go has no direct equivalent to this Sarama option

  auth:
    sasl:
      version: 1
        # Franz-go negotiates SASL handshake version automatically

# Both fields are still accepted in 0.153 for backwards compatibility.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
            </Grid>
          </Stack>
        )}

        {/* ────────── PROCESSORS ────────── */}
        {active === "processors" && (
          <Stack gap={20}>
            <H2>Processor Changes</H2>
            <Callout tone="warning" title="Error mode behavior changed in 0.153">
              The <Code>filter</Code> and <Code>transform</Code> processors now default to{" "}
              <Code>error_mode: ignore</Code>. If your pipelines relied on errors propagating
              (failing the batch), you must explicitly set <Code>error_mode: propagate</Code>.
            </Callout>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill><Text tone="secondary" style={{ fontSize: 12 }}>no config change  ·  </Text>
              <Pill tone="warning" active>P2 Degrading</Pill><Text tone="secondary" style={{ fontSize: 12 }}>config/planning required  ·  </Text>
              <Pill tone="danger" active>P1 Breaking</Pill><Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <Grid columns={2} gap={16}>
              <Card>
                <CardHeader>transform Processor — Mixed Config Style (0.124) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>transform</Code> processor now requires exactly one configuration
                      style per processor block. <strong>Basic Config</strong> (signal-specific keys
                      like <Code>metric_statements:</Code>) and <strong>Advanced Config</strong>{" "}
                      (top-level <Code>statements:</Code>) cannot be mixed. Startup failure if both
                      are present.
                    </Callout>
                    <Code>{`# BEFORE — mixing styles causes startup failure in 0.124+:
processors:
  transform:
    statements:                 # Advanced Config
      - set(attributes["env"], "prod")
    metric_statements:          # Basic Config — CANNOT mix
      - context: metric
        statements:
          - set(name, Concat([name, "_v2"], ""))

# AFTER — choose ONE style:
processors:
  transform:
    metric_statements:          # Pure Basic Config (recommended)
      - context: metric
        statements:
          - set(name, Concat([name, "_v2"], ""))
    log_statements:
      - context: log
        statements:
          - set(attributes["env"], "prod")`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>k8sattributes node_from_env_var Error (0.125) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      If <Code>node_from_env_var</Code> references an environment variable that is
                      not set, the collector now fails to start. Before 0.125 it silently monitored
                      the entire cluster instead of a single node.
                    </Callout>
                    <Code>{`# Ensure the env var is always injected (Kubernetes Downward API):
# In your pod spec:
#   env:
#     - name: MY_NODE_NAME
#       valueFrom:
#         fieldRef:
#           fieldPath: spec.nodeName

processors:
  k8s_attributes:
    node_from_env_var: MY_NODE_NAME   # safe when env var is always set

# To intentionally monitor whole cluster, omit node_from_env_var:
processors:
  k8s_attributes: {}   # monitors full cluster`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>routingprocessor Removed (0.134) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>routingprocessor</Code> was removed in 0.134. Migrate to the{" "}
                      <Code>routing</Code> connector. The connector requires a complete pipeline
                      rewiring — just swapping the component definition is not enough and will
                      still cause a startup failure.
                    </Callout>
                    <Code>{`# BEFORE — processor (removed 0.134):
processors:
  routing:
    from_attribute: X-Tenant
    table:
      - value: acme
        exporters: [otlp/acme]
      - value: beta
        exporters: [otlp/beta]

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch, routing]
      exporters: [otlp/acme, otlp/beta]

# AFTER — connector with FULL pipeline rewiring:
# NOTE: match_once removed in 0.120 — do not include it
connectors:
  routing:
    table:
      - condition: attributes["X-Tenant"] == "acme"
        pipelines: [traces/acme]
      - condition: attributes["X-Tenant"] == "beta"
        pipelines: [traces/beta]
    default_pipelines: [traces/default]

service:
  pipelines:
    traces/input:          # connector is the EXPORTER here
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [routing]

    traces/acme:           # connector is the RECEIVER here
      receivers: [routing]
      exporters: [otlp/acme]

    traces/beta:
      receivers: [routing]
      exporters: [otlp/beta]

    traces/default:
      receivers: [routing]
      exporters: [otlp/default]`}</Code>
                    <Text tone="secondary">
                      Also remove any other processors that were removed in this range:
                      <Code>datadogsemantics</Code> (0.148) and DNS lookup processor skeleton (0.151).
                    </Text>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>filter / transform — Error Mode Default <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# To restore previous propagate behavior:
processors:
  filter:
    error_mode: propagate   # was the default before 0.153

  transform:
    error_mode: propagate   # was the default before 0.153

# Or to explicitly adopt the new default:
processors:
  filter:
    error_mode: ignore

  transform:
    error_mode: ignore

# To revert via feature gate:
# --feature-gates=-processor.filter.defaultErrorModeIgnore
# --feature-gates=-processor.transform.defaultErrorModeIgnore`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Batch Processor Telemetry — level: normal Required (0.122) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Batch processor metrics are no longer emitted at <Code>level: basic</Code>{" "}
                      telemetry verbosity. Switch to <Code>level: normal</Code> to keep seeing
                      batch processor metrics.
                    </Callout>
                    <Code>{`# BEFORE — batch metrics visible at basic (pre-0.122):
service:
  telemetry:
    metrics:
      level: basic

# AFTER — switch to normal to restore batch metrics:
service:
  telemetry:
    metrics:
      level: normal   # default if not specified`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>cumulativetodelta Processor <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      Default <Code>max_staleness</Code> changed from <Code>0</Code> (infinite) to{" "}
                      <Code>1h</Code> in 0.142 to prevent unbounded memory growth.
                    </Text>
                    <Code>{`processors:
  cumulativetodelta:
    # Explicitly set to restore infinite retention:
    max_staleness: 0

    # The new default if you don't set it:
    # max_staleness: 1h`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>resourcedetection Processor <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text weight="semibold">Removed config and gates:</Text>
                    <Code>{`# REMOVED in 0.142 - use 'resource' processor instead:
processors:
  resourcedetection:
    attributes: [cloud.region]  # REMOVED field

# Feature gates removed (now always on):
# processor.resourcedetection.propagateerrors
# processor.resourcedetection.removeGCPFaasID`}</Code>
                    <Text weight="semibold" style={{ marginTop: 8 }}>Cloud platform value changes (0.147):</Text>
                    <Code>{`# Before 0.147:
#   cloud.platform: azure_eks
#   cloud.platform: azure_vm

# After 0.147 (aligns with OTel semconv v1.39):
#   cloud.platform: azure.eks
#   cloud.platform: azure.vm

# GCP FaaS attribute replaced:
#   faas.id  →  faas.instance`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>tail_sampling Processor <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      The <Code>disableinvertdecisions</Code> gate was stabilized in 0.152. Invert
                      decisions are permanently disabled. Migrate to drop policies.
                    </Text>
                    <Code>{`# OLD (no longer works after 0.144/0.152):
#   invert_match: true  on any policy

# NEW: use a dedicated drop policy instead of
# inverting an existing sampling policy.
# See the tail_sampling processor documentation
# for drop policy syntax.

# Remove this flag:
# --feature-gates=processor.tailsamplingprocessor.disableinvertdecisions`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>k8s_attributes Processor <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>New semantic convention feature gates in 0.146 (both alpha, disabled by default):</Text>
                    <Code>{`# New gates (alpha, opt-in):
# processor.k8sattributes.EmitV1K8sConventions
#   → enables k8s.<type>.label.<name> (singular)
# processor.k8sattributes.DontEmitV0K8sConventions
#   → disables k8s.<type>.labels.<name> (plural)

# allowLabelsAnnotationsSingular gate (0.135) deprecated in 0.146.

# otelcol.k8s.pod.association metric DISABLED by default (0.151):
# Disabled until pod_identifier attribute is properly calculated.
# No config change required — informational only.

# Renamed processor (alias still works):
processors:
  k8s_attributes: {}   # was: k8sattributes`}</Code>
                  </Stack>
                </CardBody>
              </Card>
            </Grid>
          </Stack>
        )}

        {/* ────────── RECEIVERS ────────── */}
        {active === "receivers" && (
          <Stack gap={20}>
            <H2>Receiver Changes</H2>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill><Text tone="secondary" style={{ fontSize: 12 }}>no config change  ·  </Text>
              <Pill tone="warning" active>P2 Degrading</Pill><Text tone="secondary" style={{ fontSize: 12 }}>config/planning required  ·  </Text>
              <Pill tone="danger" active>P1 Breaking</Pill><Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <Grid columns={2} gap={16}>
              <Card>
                <CardHeader>sqlserver Receiver — Event Flag Renames (0.128) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      The flags for enabling top query and query sample collection were renamed in
                      0.128. The old names cause a startup failure.
                    </Text>
                    <Code>{`# Before 0.128 (removed):
receivers:
  sqlserver:
    top_query_collection:
      enabled: true
    query_sample_collection:
      enabled: true

# After 0.128:
receivers:
  sqlserver:
    events:
      "db.server.top_query":
        enabled: true
      "db.server.query_sample":
        enabled: true`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>postgresql Receiver — Query Collection Flags Removed (0.132) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      The opt-in flags for query collection were removed. Both collections are now
                      enabled by default. This may increase data volume for PostgreSQL users.
                    </Callout>
                    <Code>{`# REMOVED in 0.132:
receivers:
  postgresql:
    query_sample_collection:
      enabled: true/false   # REMOVED — always enabled now
    top_query_collection:
      enabled: true/false   # REMOVED — always enabled now

# If you need to disable, use filter processor or
# the events config in newer versions.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kubelet_stats — No-Op Config Sections Removed (0.136) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      If your kubeletstats config contains sections for metrics that were already
                      disabled, the collector will fail to start after upgrading to 0.136+. The
                      Splunk distribution's converter that silently handled these no-op sections was
                      removed.
                    </Callout>
                    <Code>{`# Example of a no-op section that now causes failure:
receivers:
  kubeletstats:
    metrics:
      k8s.node.cpu.utilization:
        enabled: false   # This was already the default
      k8s.pod.cpu.utilization:
        enabled: false   # This was already the default
# ACTION: Remove any explicit 'enabled: false' sections
# for metrics that are already disabled by default.
# Run: otelcol validate --config=config.yaml to check.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kafka Receiver <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# Removed in 0.141/0.147:
receivers:
  kafka:
    topic: my-topic        # REMOVED — use topics: [list]
    exclude_topic: bad     # REMOVED — use exclude_topics: [list]
    default_fetch_size: …  # REMOVED (Sarama-only)

# Correct form:
receivers:
  kafka:
    topics: [my-topic]
    exclude_topics: [bad]`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>mysql / postgresql Receivers (0.148) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# mysql: query_sample now off by default (0.148)
# Re-enable if needed in config.

# postgresql: top_query and query_sample off by default
receivers:
  postgresql:
    # To re-enable:
    top_queries:
      enabled: true
    query_sample:
      enabled: true`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>prometheus Receiver <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text weight="semibold">Removed config options:</Text>
                    <Code>{`# Removed in 0.143:
receivers:
  prometheus:
    use_start_time_metric: true          # REMOVED
    start_time_metric_regex: ".*_start"  # REMOVED
    # Use the metricstarttime processor instead

# Removed in 0.149:
    report_extra_scrape_metrics: true    # REMOVED
    # Use PromConfig.ScrapeConfigs.ExtraScrapeMetrics`}</Code>
                    <Text weight="semibold" style={{ marginTop: 8 }}>Feature gates removed (all now always on):</Text>
                    <Code>{`# Remove if set on startup:
# receiver.prometheusreceiver.EnableNativeHistograms
# receiver.prometheusreceiver.RemoveStartTimeAdjustment
# receiver.prometheusreceiver.UseCreatedMetric
# receiver.prometheusreceiver.RemoveLegacyResourceAttributes
# receiver.prometheusreceiver.RemoveReportExtraScrapeMetricsConfig`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>prometheus Receiver — Metric Name Dots (0.120) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Prometheus 3.0 (adopted in 0.120) no longer escapes dots in metric names.
                      Internal metrics previously using underscores now use dots. Dashboard queries
                      and alert rules using the old names will break.
                    </Callout>
                    <Code>{`# Key internal metric renames (0.120+):
# processor_filter_datapoints_filtered
#   → processor_filter_datapoints.filtered
# processor_filter_logs_filtered
#   → processor_filter_logs.filtered
# deltatocumulative_streams_tracked
#   → deltatocumulative.streams.tracked
# deltatocumulative_datapoints_processed
#   → deltatocumulative.datapoints.processed

# Resource attribute renames (self-monitoring scrapes):
# service_name         → service.name
# service_instance_id  → service.instance.id
# service_version      → service.version

# Action: update all dashboards, alert rules, and
# OTTL conditions referencing the old underscore names.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>activedirectoryds Receiver — Typo Fix (0.120) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      A typo in the attribute name was fixed: <Code>distingushed_names</Code>{" "}
                      (missing 'i') → <Code>distinguished_names</Code>. OTTL rules, dashboards, and
                      filter conditions using the old misspelled name will silently stop matching.
                    </Callout>
                    <Code>{`# Search all configs and dashboards for the misspelling:
# OLD: attributes["distingushed_names"]
# NEW: attributes["distinguished_names"]
#         ↑ note the 'i' added here`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>sqlserver Receiver — db.lock_timeout Unit (0.124) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      The <Code>db.lock_timeout</Code> attribute unit changed from milliseconds to
                      seconds in 0.124. Alert thresholds and dashboard visualizations will be off by
                      a factor of 1000 after upgrade.
                    </Callout>
                    <Code>{`# No config change required.
# Action: divide all db.lock_timeout alert thresholds
# and dashboard axis values by 1000 after upgrading.

# Example: old threshold "5000" (5s in ms) → new "5" (5s in seconds)`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>sqlserver Receiver — Attributes to Resource (0.125) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Three attributes moved from <strong>log record attributes</strong> to{" "}
                      <strong>resource attributes</strong>. Log parsing and dashboards reading these
                      from log attributes will stop working.
                    </Callout>
                    <Code>{`# Deprecated log attributes (0.125+):
# attributes["computer_name"]     → resource.attributes["sqlserver.computer.name"]
# attributes["instance_name"]     → resource.attributes["sqlserver.instance.name"]
# attributes["host.name"]         → resource.attributes["host.name"]

# Update OTTL, dashboards, and log parsers to read
# from resource attributes instead of log attributes.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kubeletstats — CPU Utilization → CPU Usage Migration (0.125) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Deprecated CPU utilization metrics will be replaced by CPU usage metrics.
                      Gate promoted to beta in 0.125, stable in 0.130.
                    </Callout>
                    <Code>{`# Deprecated (being replaced):
# container.cpu.utilization → container.cpu.usage
# k8s.pod.cpu.utilization   → k8s.pod.cpu.usage
# k8s.node.cpu.utilization  → k8s.node.cpu.usage

# To keep deprecated metrics during transition:
receivers:
  kubelet_stats:
    metrics:
      container.cpu.utilization:
        enabled: true
      k8s.pod.cpu.utilization:
        enabled: true
      k8s.node.cpu.utilization:
        enabled: true
# Gate to disable: -receiver.kubeletstats.enableCPUUsageMetrics
# WARNING: gate is STABLE from 0.130 — this flag FAILS collector startup from 0.130+`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>prometheus Receiver — Legacy Resource Attributes Renamed (0.126) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      <strong>Silent dashboard/alert breakage.</strong> The
                      <Code>RemoveLegacyResourceAttributes</Code> gate became beta (enabled by default)
                      in 0.126. Resource attributes on all prometheus-scraped metrics are renamed:
                      <Code>net.host.name</Code> → <Code>server.address</Code>,{" "}
                      <Code>net.host.port</Code> → <Code>server.port</Code>,{" "}
                      <Code>http.scheme</Code> → <Code>url.scheme</Code>. Gate became STABLE in
                      0.129 — permanent from 0.129+.
                    </Callout>
                    <Code>{`# Temporary rollback (0.126–0.128 ONLY — gate is stable/locked from 0.129+):
# --feature-gates=-receiver.prometheusreceiver.RemoveLegacyResourceAttributes

# Required: update dashboards and alerts to use new attribute names:
# net.host.name  →  server.address
# net.host.port  →  server.port
# http.scheme    →  url.scheme`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>sqlserver Receiver — Event Name Changes (0.126) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      Event names and properties changed in 0.126. Any OTTL rules, dashboards, or
                      downstream processing that reference the old names will break.
                    </Text>
                    <Code>{`# Event name changes:
# "top query"    →  "db.server.top_query"
# "query sample" →  "db.server.query_sample"

# Attribute rename:
# sqlserver.username  →  user.name (in query sample events)

# Query sample body removed entirely.
# Data previously in the body is now in attributes.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>prometheus Receiver — Start Time No Longer Adjusted (0.140) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      From 0.140 onwards, the prometheus receiver no longer adjusts the start time
                      of metrics using <Code>process_start_time_seconds</Code>. The
                      <Code>RemoveStartTimeAdjustment</Code> feature gate was stabilized in 0.142
                      and removed in 0.151.
                    </Text>
                    <Code>{`# If you need start time adjustment, use the
# metricstarttime processor instead:
processors:
  metricstarttime: {}

# Feature gates that are now permanently removed:
# receiver.prometheusreceiver.RemoveStartTimeAdjustment (stable 0.142, removed 0.151)
# receiver.prometheusreceiver.UseCreatedMetric (removed 0.151)
# receiver.prometheusreceiver.EnableNativeHistograms (removed 0.151)`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>docker_observer / dockerstats — Docker API Upgrade (0.141 / 0.142) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      Two separate receivers had their default Docker API version upgraded from 1.24
                      to 1.44. If your Docker daemon is older than 1.44, set the version explicitly.
                    </Text>
                    <Code>{`# docker_observer (receiver_creator discovery) — changed in 0.141:
extensions:
  docker_observer:
    api_version: "1.24"   # add this to pin to older version

# docker_stats receiver — changed in 0.142:
receivers:
  docker_stats:
    api_version: "1.24"   # add this to pin to older version

# Both receivers default to Docker API 1.44 from these versions on.
# Minimum supported API version has NOT changed; only the default.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>mongodb Receiver (0.147) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# Schema change — single resource per server now:
# BEFORE: each database = separate resource
# AFTER:  single resource per MongoDB server

# 'database' resource attribute REMOVED.
# Database is now a metric-level attribute: db.namespace

# Added: service.instance.id resource attribute
# (UUID v5 derived from host:port)

# Update any dashboards that joined on the
# 'database' resource attribute.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>windows_event_log Receiver (0.148) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text weight="semibold">event_data format change:</Text>
                    <Code>{`# Before 0.148:
# body["event_data"] = [{"ProcessId": "1234"}]

# After 0.148+:
# body["event_data"]["ProcessId"] = "1234"

# To restore previous array format:
receivers:
  windows_event_log:
    event_data_format: array`}</Code>
                    <Text weight="semibold" style={{ marginTop: 8 }}>New body keys also added (0.148):</Text>
                    <Code>{`# RenderingInfo is now also emitted as:
# body["rendering_info"]["culture"]
# body["rendering_info"]["channel"]
# body["rendering_info"]["provider"]
# body["rendering_info"]["message"]

# UserData (alternative to EventData) now emitted as:
# body["user_data"][...] (parsed key-value map)

# These are NEW keys — not previously present.
# Update any OTTL/filter rules that inspect the log body.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>http_check Receiver — Nanoseconds (0.153) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Timing metrics unit changed: milliseconds → nanoseconds. A 500µs event that
                      previously showed 0ms now shows 500,000ns.
                    </Callout>
                    <Code>{`# Affected metrics (update dashboard thresholds):
# httpcheck.dns.lookup.duration
# httpcheck.client.connection.duration
# httpcheck.tls.handshake.duration
# httpcheck.client.request.duration
# httpcheck.response.duration

# Old: 500µs = 0 (truncated to 0ms)
# New: 500µs = 500,000 (nanoseconds)
# Old: 1.5ms = 1 (ms)
# New: 1.5ms = 1,500,000 (ns)`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>k8sobjects Receiver — API Check Moved (0.125) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      K8s API object existence check moved from config validation to receiver
                      startup. <Code>otelcol validate</Code> no longer catches missing K8s API
                      objects — test with a live cluster instead.
                    </Text>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kubelet_stats Receiver (0.150) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>Deprecated resource attributes now disabled by default:</Text>
                    <Code>{`# Disabled by default in 0.150+:
#   aws.volume.id
#   fs.type
#   gce.pd.name
#   glusterfs.endpoints.name
#   glusterfs.path
#   partition
# These will be removed in a future release.
# Re-enable them explicitly in config if still needed.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>jaeger Receiver <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      <Code>DisableRemoteSampling</Code> gate stabilized in 0.151 and the gate
                      itself removed in 0.153. Remote sampling is now always disabled.
                    </Text>
                    <Code>{`# Remove this flag if present:
# --feature-gates=receiver.jaeger.DisableRemoteSampling

# If you relied on remote sampling, configure an
# alternative strategy (e.g. tail_sampling processor
# or probabilistic sampler).`}</Code>
                  </Stack>
                </CardBody>
              </Card>
            </Grid>
          </Stack>
        )}

        {/* ────────── EXPORTERS ────────── */}
        {active === "exporters" && (
          <Stack gap={20}>
            <H2>Exporter Changes</H2>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill><Text tone="secondary" style={{ fontSize: 12 }}>no config change  ·  </Text>
              <Pill tone="warning" active>P2 Degrading</Pill><Text tone="secondary" style={{ fontSize: 12 }}>config/planning required  ·  </Text>
              <Pill tone="danger" active>P1 Breaking</Pill><Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <Grid columns={2} gap={16}>
              <Card>
                <CardHeader>signalfx Exporter — translation_rules Removed (0.121) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>translation_rules</Code> option in the signalfx exporter was removed
                      in 0.121. Collector fails to start if this field is present. Replace with a
                      <Code>transform</Code> processor.
                    </Callout>
                    <Code>{`# BEFORE — causes startup failure in 0.121+:
exporters:
  signalfx:
    access_token: "..."
    translation_rules:        # REMOVE this block
      - action: rename_metrics
        mapping:
          old_metric: new_metric

# AFTER — use transform processor:
processors:
  transform:
    metric_statements:
      - context: metric
        statements:
          - set(name, "new_metric") where name == "old_metric"
exporters:
  signalfx:
    access_token: "..."`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>sending_queue::blocking Field Removed (0.129) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>blocking</Code> field inside <Code>sending_queue:</Code> was deprecated
                      in 0.123 and removed in 0.129. Collector fails to start if present. Use
                      <Code>block_on_overflow</Code> instead.
                    </Callout>
                    <Code>{`# BEFORE — startup failure in 0.129+:
exporters:
  otlp:
    sending_queue:
      blocking: true   # removed

# AFTER:
exporters:
  otlp:
    sending_queue:
      block_on_overflow: true`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>OTLP Exporter — Deprecated Batcher Removed (0.130) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      The deprecated top-level <Code>batcher</Code> config block was removed from the
                      OTLP exporter in 0.130. Use <Code>sending_queue::batch</Code> instead.
                    </Text>
                    <Code>{`# Before 0.130 (removed):
exporters:
  otlp:
    batcher:
      enabled: true
      min_size_items: 100

# After 0.130:
exporters:
  otlp:
    sending_queue:
      batch:
        min_size: 100
        flush_timeout: 200ms`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>sapm Exporter — Removed (0.147) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# BEFORE — sapm exporter (permanently removed):
exporters:
  sapm:
    endpoint: https://ingest.us0.signalfx.com/v2/trace
    access_token: "\${env:SPLUNK_ACCESS_TOKEN}"

# AFTER — migrate to otlphttp:
exporters:
  otlphttp:
    endpoint: https://ingest.us0.signalfx.com/v2/trace/otlp
    headers:
      X-SF-Token: "\${env:SPLUNK_ACCESS_TOKEN}"`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kafka Exporter — Top-Level Fields Removed (0.148) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# REMOVED (deprecated since 0.124, removed 0.148):
exporters:
  kafka:
    topic: my-topic      # REMOVED
    encoding: otlp_proto # REMOVED

# REQUIRED new form — per-signal fields:
exporters:
  kafka:
    logs:
      topic: logs-topic
      encoding: otlp_proto
    metrics:
      topic: metrics-topic
      encoding: otlp_proto
    traces:
      topic: traces-topic
      encoding: otlp_proto`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>splunk_hec Exporter — batcher Removed (0.151) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# BEFORE (removed in 0.151):
exporters:
  splunk_hec:
    batcher:
      enabled: true
      min_size_items: 100

# AFTER — migrate to sending_queue::batch:
exporters:
  splunk_hec:
    sending_queue:
      batch:
        min_size_items: 100`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>awss3 Exporter — s3_partition Removed (0.121) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      <Code>s3_partition</Code> was replaced by <Code>s3_partition_format</Code> in
                      0.121. Collector fails to start if the old key is present.
                    </Callout>
                    <Code>{`# BEFORE — removed in 0.121:
exporters:
  awss3:
    s3uploader:
      s3_partition: minute    # REMOVED

# AFTER — equivalent strftime format:
exporters:
  awss3:
    s3uploader:
      s3_partition_format: "year=%Y/month=%m/day=%d/hour=%H/minute=%M"
# s3_partition: hour → "year=%Y/month=%m/day=%d/hour=%H"
# s3_partition: day  → "year=%Y/month=%m/day=%d"`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>prometheusremotewrite — export_created_metric Removed (0.123) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      <Code>export_created_metric</Code> was removed in 0.123. Collector fails to
                      start if present.
                    </Callout>
                    <Code>{`# BEFORE — startup failure in 0.123+:
exporters:
  prometheusremotewrite:
    endpoint: https://prometheus.example.com/api/v1/write
    export_created_metric: true    # REMOVE

# AFTER:
exporters:
  prometheusremotewrite:
    endpoint: https://prometheus.example.com/api/v1/write`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>kafka Exporter + kafkametrics Receiver — Client ID Changes (0.123) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      <strong>Silent data loss risk (0.123).</strong> The default <Code>client_id</Code> for
                      <Code>kafkaexporter</Code> and <Code>kafkametricsreceiver</Code> changed from
                      <Code>"sarama"</Code> to <Code>"otel-collector"</Code>. Kafka ACLs based on
                      <Code>client.id = "sarama"</Code> will silently break. Also:
                      <Code>auth.plain_text</Code> deprecated (use <Code>auth.sasl</Code> with PLAIN);
                      <Code>refresh_frequency</Code> deprecated (use <Code>metadata.refresh_interval</Code>).
                    </Callout>
                    <Code>{`# Pin client_id to preserve ACL compatibility:
exporters:
  kafka:
    client_id: sarama

receivers:
  kafkametrics:
    client_id: sarama
    metadata:
      refresh_interval: 30s   # was refresh_frequency: 30s`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>prometheusremotewrite Exporter (0.142) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# Updated to Remote Write 2.0 rc.4.
# Requires Prometheus 3.8.0+ as receiving endpoint.
# Wire-protocol incompatible with older Prometheus.
# Upgrade Prometheus before upgrading the collector
# if exporting to a Prometheus server.

# Feature gate removed in 0.151 — remove if set:
# pkg.translator.prometheus.NormalizeName`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>signalfx Exporter — URL Change (0.151) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Default endpoint domain changed. Update firewall allowlists.
                    </Callout>
                    <Code>{`# Before 0.151 (realm: us0):
# api_url: https://api.us0.signalfx.com
# ingest_url: https://ingest.us0.signalfx.com

# After 0.151:
# api_url: https://api.us0.observability.splunkcloud.com
# ingest_url: https://ingest.us0.observability.splunkcloud.com

# Explicit api_url / ingest_url settings are unchanged.
# ACTION: Add *.observability.splunkcloud.com to allowlists
# if your rules targeted only *.signalfx.com`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>clickhouse Exporter (0.153) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# Feature gate removed — set in config instead:

# Remove from startup:
# --feature-gates=+clickhouse.json

# Set directly in config:
exporters:
  clickhouse:
    json: true`}</Code>
                  </Stack>
                </CardBody>
              </Card>
            </Grid>
          </Stack>
        )}

        {/* ────────── OTTL ────────── */}
        {active === "ottl" && (
          <Stack gap={20}>
            <H2>OTTL (OpenTelemetry Transformation Language) Changes</H2>
            <Callout tone="warning" title="Silent failures now surface as errors">
              OTTL statements that were previously silently failing due to type mismatches will now
              return errors. Review all transform/filter OTTL configurations and test in staging.
            </Callout>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill><Text tone="secondary" style={{ fontSize: 12 }}>no config change  ·  </Text>
              <Pill tone="warning" active>P2 Degrading</Pill><Text tone="secondary" style={{ fontSize: 12 }}>config/planning required  ·  </Text>
              <Pill tone="danger" active>P1 Breaking</Pill><Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <Grid columns={2} gap={16}>
              <Card>
                <CardHeader>SetMap Error Handling (0.150) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      <Code>SetMap</Code> now returns an error for invalid value types instead of
                      silently ignoring them. Slice setters also use stricter type handling.
                    </Text>
                    <Code>{`# Review all set() calls on map or slice fields
# in your transform/filter pipelines.
# Statements that were previously no-ops may now
# produce errors that propagate or get ignored
# depending on your error_mode setting.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Type-Strict Setters (0.150 + 0.153) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      OTTL context setters now validate expected types and return errors on
                      mismatch. Previously these silently no-opped.
                    </Text>
                    <Code>{`# Now returns an error (was silently ignored):
# Setting histogram-specific path on a NumberDataPoint:
set(explicit_bounds, [1.0])    # only valid on HistogramDataPoint
set(bucket_counts, [1, 2, 3])  # only valid on HistogramDataPoint

# Valid paths by data point type:
# NumberDataPoint:
#   value_double, value_int, exemplars
# HistogramDataPoint:
#   explicit_bounds, bucket_counts, count, sum, exemplars
# ExponentialHistogramDataPoint:
#   scale, zero_count, positive.*, negative.*, count, sum
# SummaryDataPoint:
#   quantile_values, count, sum`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>New OTTL Functions Available Since 0.126 <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# Coalesce (0.150): first non-nil value from list
set(attributes["user"],
  Coalesce([attributes["user.id"],
             attributes["enduser.id"],
             "unknown"]))

# Base64Encode (0.146)
set(attributes["encoded"], Base64Encode(body))

# IsInCIDR (0.146): check if IP in CIDR range
where IsInCIDR(attributes["client.ip"], "10.0.0.0/8")

# Bool (0.143): convert to boolean
set(attributes["flag"], Bool(attributes["flag_str"]))

# delete_index: remove item from array (0.145)
delete_index(attributes["items"], 0)

# TrimPrefix / TrimSuffix (0.139)
set(name, TrimPrefix(name, "prefix_"))`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>truncate_all Function — UTF-8 Safe by Default (0.148) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      <Code>truncate_all</Code> now defaults to UTF-8-safe truncation. Results may
                      be slightly shorter than the limit to avoid splitting multi-byte characters.
                    </Text>
                    <Code>{`# New default (utf8_safe: true):
truncate_all(attributes, 100)
# → may produce strings shorter than 100 bytes

# To restore old byte-level truncation:
truncate_all(attributes, 100, false)
#   third argument: utf8_safe = false`}</Code>
                  </Stack>
                </CardBody>
              </Card>
            </Grid>
          </Stack>
        )}

        {/* ────────── SPLUNK-SPECIFIC ────────── */}
        {active === "splunk" && (
          <Stack gap={20}>
            <H2>Splunk-Specific Changes</H2>
            <Row gap={10} wrap align="center">
              <Pill tone="success" active>P3 Advisory</Pill><Text tone="secondary" style={{ fontSize: 12 }}>no config change  ·  </Text>
              <Pill tone="warning" active>P2 Degrading</Pill><Text tone="secondary" style={{ fontSize: 12 }}>config/planning required  ·  </Text>
              <Pill tone="danger" active>P1 Breaking</Pill><Text tone="secondary" style={{ fontSize: 12 }}>startup failure or silent data loss</Text>
            </Row>
            <Grid columns={2} gap={16}>
              <Card>
                <CardHeader>routing Connector — match_once Removed (0.120) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>match_once</Code> parameter was removed from the routing connector
                      in 0.120. If your config uses the routing connector (post-P1-01 migration) and
                      includes <Code>match_once:</Code>, it will fail to start.
                    </Callout>
                    <Code>{`# BEFORE — startup failure in 0.120+:
connectors:
  routing:
    match_once: true    # REMOVE this field
    table:
      - condition: attributes["env"] == "prod"
        pipelines: [traces/prod]

# AFTER:
connectors:
  routing:
    table:
      - condition: attributes["env"] == "prod"
        pipelines: [traces/prod]
    default_pipelines: [traces/default]`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>service::telemetry::address Silently Ignored (0.123) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The legacy flat <Code>service: telemetry: address:</Code> field is silently
                      ignored in 0.123+ — collector starts but emits <strong>no internal telemetry
                      metrics</strong>. Gate became STABLE in 0.128; the fallback flag fails startup from 0.128+.
                    </Callout>
                    <Code>{`# BEFORE — silently ignored in 0.123+:
service:
  telemetry:
    address: 0.0.0.0:8888   # IGNORED

# AFTER — use readers: format (official migration):
service:
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: 0.0.0.0
                port: 8888
                without_scope_info: true
                without_type_suffix: true
                without_units: true

# SIMPLER ALTERNATIVE (also accepted, but itself deprecated):
# service:
#   telemetry:
#     metrics:
#       address: 0.0.0.0:8888`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>access_token_passthrough Removed (0.137) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>access_token_passthrough</Code> field was removed in 0.137 from
                      the <Code>signalfx</Code> receiver, <Code>signalfx</Code> exporter, and{" "}
                      <Code>splunk_hec</Code> exporter. The collector will <strong>fail to start</strong>{" "}
                      if the field is present anywhere in your config.
                    </Callout>
                    <Text>
                      <strong>Action:</strong> Remove all occurrences of <Code>access_token_passthrough:</Code>{" "}
                      from every receiver and exporter block. Use <Code>headers_setter</Code> extension
                      or inline <Code>headers:</Code> on the exporter for token propagation instead.
                    </Text>
                    <Code>{`# BEFORE — causes startup failure in 0.137+:
receivers:
  signalfx:
    endpoint: 0.0.0.0:9943
    access_token_passthrough: true    # REMOVE

exporters:
  signalfx:
    access_token: "\${env:SPLUNK_ACCESS_TOKEN}"
    access_token_passthrough: true    # REMOVE

  splunk_hec:
    token: "\${env:SPLUNK_HEC_TOKEN}"
    access_token_passthrough: true    # REMOVE

# AFTER — field removed entirely:
receivers:
  signalfx:
    endpoint: 0.0.0.0:9943

exporters:
  signalfx:
    access_token: "\${env:SPLUNK_ACCESS_TOKEN}"

  splunk_hec:
    token: "\${env:SPLUNK_HEC_TOKEN}"`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>FluentD Fully Removed (0.144–0.145) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">FluentD is permanently removed from all Splunk installers.</Callout>
                    <Code>{`# Removed from:
# - MSI (Windows)        0.144
# - Chocolatey           0.144
# - Standard installer   0.144
# - PowerShell script    0.145
# - RPM / DEB packages   0.145

# REPLACEMENT: use the filelog receiver
receivers:
  file_log:
    include: [/var/log/myapp/*.log]
    operators:
      - type: json_parser`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>signalfx Receiver Deprecated (0.146) → Removed (0.153) <Pill tone="danger" active>P1 Breaking</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="danger">
                      The <Code>signalfx</Code> receiver has been permanently removed from the
                      Splunk distribution in v0.153.0. If your config still uses it, the
                      collector will fail to start.
                    </Callout>
                    <Text>
                      <strong>Timeline:</strong> Formally deprecated in v0.146.0 (upstream OTel Contrib,
                      date 2026-02-13). Removed from the Splunk distribution in v0.153.0. Migrate
                      to the <Code>otlp</Code> receiver on both <Code>metrics</Code> and{" "}
                      <Code>logs</Code> pipelines.
                    </Text>
                    <Code>{`# BEFORE — signalfx receiver (REMOVED in 0.153):
receivers:
  signalfx:
    endpoint: 0.0.0.0:9943
    include_metadata: true

service:
  pipelines:
    metrics:
      receivers: [signalfx]
    logs:
      receivers: [signalfx]

# AFTER — use the OTLP receiver instead:
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

service:
  pipelines:
    metrics:
      receivers: [otlp]
    logs:
      receivers: [otlp]`}</Code>
                    <Text tone="secondary">
                      Sending agents must also be updated to emit OTLP instead of SignalFx
                      protocol. The Splunk OTel agent and most supported SDKs already support
                      OTLP natively.
                    </Text>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>confighttp HTTP Client null Fields (0.121) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      HTTP client config options <Code>max_idle_conns</Code>,{" "}
                      <Code>max_idle_conns_per_host</Code>, <Code>max_conns_per_host</Code>, and{" "}
                      <Code>idle_conn_timeout</Code> changed from nullable to integer type. Setting
                      them to <Code>null</Code> causes a YAML parse error.
                    </Callout>
                    <Code>{`# BEFORE — null caused parse error in 0.121+:
exporters:
  otlphttp:
    http_client_config:
      max_idle_conns: null         # INVALID
      idle_conn_timeout: null      # INVALID

# AFTER — use 0 for unlimited, or omit:
exporters:
  otlphttp:
    http_client_config:
      max_idle_conns: 0            # 0 = unlimited
      # idle_conn_timeout omitted = 0 (default)`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>splunkenterprise Receiver — Metrics Now Opt-In (0.124) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      All <Code>splunkenterprise</Code> receiver metrics are now disabled by default
                      except <Code>splunk.health</Code>. If you relied on the default metric set,
                      those metrics will silently stop after upgrade.
                    </Callout>
                    <Code>{`# Explicitly re-enable each metric you need:
receivers:
  splunkenterprise:
    endpoint: https://splunk:8089
    metrics:
      splunk.data.indexes.extended.bucket.count:
        enabled: true
      splunk.index.size.extended:
        enabled: true
      splunk.license.index.usage:
        enabled: true
      # Add each metric — only splunk.health is on by default`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Kafka auth::tls Deprecated (0.124) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      The <Code>auth: tls:</Code> nested path in Kafka receiver/exporter is
                      deprecated in 0.124 in favor of a top-level <Code>tls:</Code> field. Migrate
                      to avoid a future startup failure.
                    </Callout>
                    <Code>{`# BEFORE — deprecated path (will be removed):
receivers:
  kafka:
    auth:
      tls:
        ca_file: /etc/ssl/ca.pem
        cert_file: /etc/ssl/cert.pem
        key_file: /etc/ssl/key.pem

# AFTER — top-level tls: block:
receivers:
  kafka:
    tls:
      ca_file: /etc/ssl/ca.pem
      cert_file: /etc/ssl/cert.pem
      key_file: /etc/ssl/key.pem`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>otelcol.component.kind Values Now Lowercase (0.125) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      The <Code>otelcol.component.kind</Code> attribute in internal collector
                      telemetry changed from title case to lowercase in 0.125. Any OTTL conditions
                      or dashboard queries using the old capitalized values will stop matching.
                    </Callout>
                    <Code>{`# OLD (pre-0.125):
# attributes["otelcol.component.kind"] == "Receiver"
# attributes["otelcol.component.kind"] == "Exporter"
# attributes["otelcol.component.kind"] == "Processor"

# NEW (0.125+):
# attributes["otelcol.component.kind"] == "receiver"
# attributes["otelcol.component.kind"] == "exporter"
# attributes["otelcol.component.kind"] == "processor"`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>IIS Application Pool Metrics Enabled by Default (0.131) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Callout tone="warning">
                      Windows IIS users: two new metrics are now enabled by default. If you have
                      many application pools, this may significantly increase metric volume and
                      billing.
                    </Callout>
                    <Code>{`# Now enabled by default in 0.131:
# iis.application_pool.state
# iis.application_pool.uptime

# To disable:
receivers:
  iis:
    metrics:
      iis.application_pool.state:
        enabled: false
      iis.application_pool.uptime:
        enabled: false`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Windows MSI Download URL (0.151) <Pill tone="warning" active>P2 Degrading</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Code>{`# OLD URL in installer scripts:
# dl.signalfx.com

# NEW URL:
# dl.observability.splunkcloud.com

# Update any automation scripts, pipelines, or
# infrastructure-as-code that reference the old URL.`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>Config Debug Endpoint Removed (0.142) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      The localhost config endpoint <Code>http://localhost:55554/debug/configz</Code>{" "}
                      is removed. Use the <Code>zpages</Code> extension instead.
                    </Text>
                    <Code>{`# Replacement via zpages extension:
# http://localhost:55679/debug/expvarz

# Get effective config (bash):
curl http://localhost:55679/debug/expvarz --silent \
  | jq -r '.["splunk.config.effective"]'

# Get initial config (bash):
curl http://localhost:55679/debug/expvarz --silent \
  | jq -r '.["splunk.config.initial"]'

# PowerShell:
(Invoke-WebRequest http://localhost:55679/debug/expvarz).Content \
  | ConvertFrom-Json \
  | Select-Object -ExpandProperty "splunk.config.effective"`}</Code>
                  </Stack>
                </CardBody>
              </Card>
              <Card>
                <CardHeader>splunk_otlp_histograms Converter Removed (0.148) <Pill tone="success" active>P3 Advisory</Pill></CardHeader>
                <CardBody>
                  <Stack gap={10}>
                    <Text>
                      The config converter that automatically added the{" "}
                      <Code>splunk_otlp_histograms</Code> resource attribute is removed.
                    </Text>
                    <Code>{`# If you still need this attribute, add it manually:
processors:
  resource:
    attributes:
      - key: splunk_otlp_histograms
        value: "true"
        action: upsert`}</Code>
                  </Stack>
                </CardBody>
              </Card>
            </Grid>

            <H2>Removed SmartAgent Monitors — All Versions</H2>
            <Table
              headers={["Removed Monitor / Plugin", "Recommended Replacement", "Removed In"]}
              rows={REMOVED_MONITORS}
              striped
              stickyHeader
            />
          </Stack>
        )}

        {/* ────────── CHECKLIST ────────── */}
        {active === "checklist" && (
          <Stack gap={20}>
            <H2>Step-by-Step Upgrade Checklist</H2>
            <Text tone="secondary">
              Work through each group in order. Complete all items before moving to the next group.
            </Text>
            {CHECKLIST.map((group) => (
              <CollapsibleSection key={group.group} title={group.group} defaultOpen>
                <Stack gap={8} style={{ paddingLeft: 8, paddingTop: 8 }}>
                  {group.items.map((item, i) => (
                    <Row key={i} gap={10} align="start">
                      <Text
                        tone="tertiary"
                        style={{ minWidth: 22, fontVariantNumeric: "tabular-nums" }}
                      >
                        {i + 1}.
                      </Text>
                      <Text>{item}</Text>
                    </Row>
                  ))}
                </Stack>
              </CollapsibleSection>
            ))}

            <Divider />

            <Card>
              <CardHeader>Recommended Upgrade Strategy</CardHeader>
              <CardBody>
                <Stack gap={12}>
                  <Text>
                    Given the size of this upgrade, use a phased approach with a staging environment.
                  </Text>
                  <Grid columns={3} gap={12}>
                    <Stack gap={6}>
                      <Pill tone="info" active>Phase 1 — Config Prep</Pill>
                      <Text>
                        Update all component names to canonical snake_case. Remove all removed config
                        keys. Test config validity with{" "}
                        <Code>otelcol validate --config=config.yaml</Code>.
                      </Text>
                    </Stack>
                    <Stack gap={6}>
                      <Pill tone="warning" active>Phase 2 — Staging</Pill>
                      <Text>
                        Deploy to staging. Validate pipelines, metrics flow, and no unexpected errors
                        from OTTL type changes or error_mode changes. Monitor for at least 24 hours.
                      </Text>
                    </Stack>
                    <Stack gap={6}>
                      <Pill tone="success" active>Phase 3 — Production</Pill>
                      <Text>
                        Deploy with rolling update. Update firewall allowlists for new SignalFx URL.
                        Update dashboards for unit and label changes. Remove old feature gate flags.
                      </Text>
                    </Stack>
                  </Grid>
                </Stack>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>Validation Commands</CardHeader>
              <CardBody>
                <Code>{`# Validate your config before upgrading:
otelcol validate --config=/etc/otelcol/config.yaml

# Check for deprecated component name warnings on startup:
otelcol --config=/etc/otelcol/config.yaml 2>&1 | grep -i "deprecated"

# Linux: verify collector service health post-upgrade:
systemctl status splunk-otel-collector

# Windows: check service status (PowerShell):
Get-Service -Name splunk-otel-collector | Select-Object Status, StartType

# View effective config via zpages:
curl http://localhost:55679/debug/expvarz | jq -r '.["splunk.config.effective"]'`}</Code>
              </CardBody>
            </Card>
          </Stack>
        )}

        <Spacer />
      </Stack>
    </Stack>
  );
}
