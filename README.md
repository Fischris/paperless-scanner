# paperless-scanner
A paperless scanner image for EPSON like ET-4950 for trigger via manual Homeassistant 

Some Epson Scanner (like my ET-4950) have no Scan-to-SMB. This is a way to "implement" it.

I do not want to use either a smartphone-based workflow or Epson's cloud/email workaround.

I use Home Assistant with an Aqara H1 double remote switch:
Left button: trigger /scan/adf
Right button: trigger /scan/flatbed

This image exposes these endpoints and writes the scanned files to the configured target directory.
