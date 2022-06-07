# pebble-log-parser
Stream logs from Pebble workloads while stripping out their own logfmt to sanitize it

Connect to Pebble and pull streaming logs for all client workloads. Sift out known
logging formats (`logfmt`, for example), add some additional labels with `zap`, and
send them along. First step to making them ingestable with Loki in a sane manner
