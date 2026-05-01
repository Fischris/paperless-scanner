# paperless-scanner
A paperless scanner image for EPSON like ET-4950 for trigger via manual Homeassistant 

Some Epson Scanner (like my ET-4950) have no Scan-to-SMB. This is a way to "implement" it.

I don't want neither use my smartphone nor the Epson-Server for an E-mail workaround.

i am using Homeassistant. The idea is to use an Aqara H1 double remote switch. Left click will call /scan/adf. Right click will call /scan/flatbed.

This image will provide those endpoints and put the scanned files in the provided dir.
