# prometheus-natureremo-exporter
Prometheus exporter for Nature Remo


## Remarks
prometheus-natureremo-exporter makes two API calls per scrape. It is recommended to either 1) set `scrape_interval` to 1m, or 2) set up a caching proxy and specify `--natureremo.endpoint https://caching-proxy` while keeping `scrape_interval` to a lower value.
