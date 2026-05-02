# paperless-scanner
A paperless scanner image for EPSON like ET-4950 for trigger via manual Homeassistant 

Some Epson Scanner (like my ET-4950) have no Scan-to-SMB. This is a way to "implement" it.

I do not want to use either a smartphone-based workflow or Epson's cloud/email workaround.

I use Home Assistant with an Aqara H1 double remote switch:
- Left button: trigger /scan/adf
- Right button: trigger /scan/flatbed

This image exposes these endpoints and writes the scanned files to the configured target directory.

## Endpoints

- `GET /healthz`
- `POST /scan/adf`
- `POST /scan/flatbed`

`/scan/adf` and `/scan/flatbed` require an `Authorization` header in the following format:

```text
Authorization: Bearer <AUTH_TOKEN>
```

## Required environment variables

The following environment variables must be provided when running the container:

- `TARGET_DIR`
  Directory where scanned files will be written.

- `AUTH_TOKEN`
  Bearer token required for `/scan/adf` and `/scan/flatbed`.

- `SCANNER_DEVICE`
  The scanner device name used by `scanimage`.

If `SCANNER_DEVICE` is not set, the service will run `scanimage -L`, print the discovered devices to the logs and then exit.

## Optional environment variables

- `PORT`  
  HTTP port for the service. Default: `8080`

- `SCAN_RESOLUTION`  
  Scan resolution in DPI. Default: `300`

  ## Example with Docker

```bash
docker run --rm \
  -p 8080:8080 \
  -e TARGET_DIR=/scans \
  -e AUTH_TOKEN=change-this-secret-token \
  -e SCANNER_DEVICE="airscan:e0:EPSON ET-4950" \
  -e SCAN_RESOLUTION=300 \
  -v /path/to/scans:/scans \
  ghcr.io/fischris/paperless-scanner:latest
```
## Example requests

wip


  
