# Goncho Python examples

These examples use Goncho's stable local HTTP/server API from Python's standard library.

Start a local server first:

```bash
goncho-server serve -db ./goncho.db -addr 127.0.0.1:8765
```

Then run:

```bash
python3 ./examples/python/http_recall.py
```

The example calls the loopback `/v3/workspaces/{workspace}/peers/{peer}/recall` endpoint. It does not install connectors or mutate host configuration.
