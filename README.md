# zcan
Access Zehnder data via CAN interface

## What?
This small module and app was written to interact with a Zehnder ComfoQ system via the CAN bus. It uses the socketcan interface to send and receive packets via the CAN bus. If your CAN module does not use the socketcan interface this module isn't for you.

## Why?
As part of our Zehnder installation we have a ComfoConnect LAN-C module which allows access via our home network. This works well and we use the Zehnder app to control the unit. However, we also use HomeAssistant to monitor the unit - also via the network. As only one device can be connected via the network at once, this leads to times when we loose monitoring or have difficulty in using the app. By using the second CAN interface on the LAN-C unit we should be able to retrieve the monitoring data without using the network connection. 

## Sample Output
The sample application produces a simple dump of PDO's seen on the CAN interface.

```
$ ./zcan -interface can0


Processing CAN packets. CTRL+C to quit...

Test #1 -> Serial Number: SITxxxxxxxx
Test #2 -> Model DEscription: ComfoAir Q450 GB ST ERV
Test #3 - Multiple
	Serial Number: SITxxxxxxxx
	Version: 3222284288 -> [3.1]
	Model: ComfoAir Q450 GB ST ERV
^C
122: Supply Fan Speed                             : 0xC408        2244 rpm
121: Exhaust Fan Speed                            : 0x0508        2053 rpm
117: Exhaust Fan Duty                             : 0x37            55 %
118: Supply Fan Duty                              : 0x3B            59 %
213: Avoided Heating: Actual                      : 0x5A02       30.10 W
119: Exhaust Fan Flow                             : 0xFA00         250 m³/h
305: Unknown sensor 305                           : 0x4500          69 unknown
128: Power Consumption                            : 0x3700          55 W
```
