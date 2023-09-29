# zcan
Access Zehnder data via CAN interface

## What?
This small module and app was written to interact with a Zehnder ComfoQ system via the CAN bus. It uses the socketcan interface to send and receive packets via the CAN bus. If your CAN module does not use the socketcan interface this module isn't for you.

## Why?
As part of our Zehnder installation we have a ComfoConnect LAN-C module which allows access via our home network. This works well and we use the Zehnder app to control the unit. However, we also use HomeAssistant to monitor the unit - also via the network. As only one device can be connected via the network at once, this leads to times when we loose monitoring or have difficulty in using the app. By using the second CAN interface on the LAN-C unit we should be able to retrieve the monitoring data without using the network connection. 

## Sample Output
The sample application produces a simple dump of PDO's seen on the CAN interface.

```
$ ./zcan -interface can0 -address 10.0.73.xxx
2023/09/28 04:38:49 Starting network services


Processing CAN packets. CTRL+C to quit...

2023/09/28 04:38:49 Starting HTTP server listening @ http://10.0.73.xxx:7004/
Test #1 -> Serial Number: SITxxxxxxxx
Test #2 -> Model DEscription: ComfoAir Q450 GB ST ERV
2023/09/28 04:38:49 Processing data for ComfoAir Q450 GB ST ERV [SITxxxxxxxx] Version 3.1
^C2023/09/29 02:06:11 HTTP server shutdown

ID   Name                                         Raw Data     Value Units
---- -------------------------------------------- ---------- ------- ---------
213  Avoided Heating Actual                       0x9F03       46.35 W
117  Exhaust Fan Duty                             0x38            56 %
119  Exhaust Fan Flow                             0xFB00         251 m³/h
121  Exhaust Fan Speed                            0x3D08        2109 rpm
128  Power Consumption                            0x3800          56 W
118  Supply Fan Duty                              0x3C            60 %
120  Supply Fan Flow                              0xFA00         250 m³/h
122  Supply Fan Speed                             0xB708        2231 rpm
222  Unknown sensor 222                           0x00             0 unknown
305  Unknown sensor 305                           0x4600          70 unknown
306  Unknown sensor 306                           0x4500          69 unknown
```

The HTTP server provides a simple JSON output of the PDO data collected.

```
{
  "avoided-heating:-actual": 33.90, 
  "supply-fan-duty": 61, 
  "unknown-sensor-222": 0, 
  "unknown-sensor-305": 70, 
  "exhaust-fan-speed": 2064, 
  "unknown-sensor-306": 71, 
  "unknown-sensor-294": 61, 
  "supply-fan-speed": 2305, 
  "exhaust-fan-duty": 54, 
  "exhaust-fan-flow": 251, 
  "supply-fan-flow": 253, 
  "unknown-sensor-418": 0, 
  "power-consumption": 58
}
```

It is possible to have the app capture the frame data and then process it. By default simply passing the -capture flag will result in a file called output being created which will contain each frame on a seperate line. This can be changed by using the -capture-filename and passing the desired filename.

```
$ ./zcan -interface can0 -output 
```

Processing the data then requires running again using the -dumpfile option.

```
$ ./zcan -dumpfile output
Processing dumpfile: output
File: output. Total size is 13822 bytes
..................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................................

RMI Messages
1F004DC1 : Source 1 Dest 55 Counter 0  Sequence 0
         : IsMulti true  IsRequest false  IsError false
         : 12 bytes [83 73 84 ...
```

## Building
When building on a RaspberryPi with the 64-bit OS, I had to set the GOARCH target to arm64 in order to build.

```
$ go env -w GOARCH=arm64
```


## Future Plans
- discover the PDO meanings for unknown sensors.  The excellent data provided by https://github.com/michaelarnauts/aiocomfoconnect/blob/master/docs/PROTOCOL-PDO.md doesn't seem to fully align with what I am seeing.
- add the ability to change settings on the unit via HTTP.
- improve the logging

