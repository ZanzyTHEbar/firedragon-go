# Client Side Architecture

## Overview

## Pre-requisites

### HID

[`libusb/hidapi`](https://github.com/libusb/hidapi) is a cross-platform library that provides an API to access Human Interface Devices (HID). It is used to capture HID events from the user's USB and/or Bluetooth HID device.

This lib (and udev rules on POSIX compliant systems) must be installed for the client to function properly. A proper application will link-against this library and use it to capture HID events in the local binary.

You **must** install the pre-requisites for the `hidapi` library **before** installing the go client library. Please read the documentation for the `hidapi` library for more information.

I would write a script to automate this - but that is outside the scope of this work-trial project.

I am using the [`go-hid`](https://github.com/sstallion/go-hid) library to interface with the `hidapi` library.

The tool `lshid` is used for debugging and development.

```bash
go install github.com/sstallion/go-hid/cmd/lshid@latest
```

### Setting up permissions for HID devices on Linux

On Linux systems, you may encounter "Permission denied" errors when trying to access HID devices. This is because by default, only root has access to these devices. To fix this, you need to create a udev rule to grant your user permission to access these devices.

1. Create a new udev rules file:

```bash
sudo nano /etc/udev/rules.d/99-hidraw-permissions.rules
```

2. Add the following rules to the file:

```
# Grant permission to all hidraw devices
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", MODE="0666"

# Alternative: Grant permission to specific user/group
# KERNEL=="hidraw*", SUBSYSTEM=="hidraw", GROUP="plugdev", MODE="0660"
```

3. Reload the udev rules:

```bash
sudo udevadm control --reload-rules
sudo udevadm trigger
```

4. You may need to reconnect your HID devices or reboot your system for the changes to take effect.

If you're still having issues, you can temporarily test with sudo (not recommended for production):

```bash
sudo ./your_application
```

### NATs

In order to run the client, you must have a NATs server running. You can install the NATs cli (which has an embedded dev server) from their [github](https://github.com/nats-io/natscli).

You can install the raw binary from their [releases](https://github.com/nats-io/natscli/releases).

However, installing via the go cli is the easiest way to get started.

## Getting Started

`Starting the NATs broker with JetStream enabled`

```bash
nats server run --jetstream --clean
```

`Start the client`

```bash
go run client/cmd/client.go -nats "nats://localhost:<port from nats_development context>" -nats-user local -nats-pass <pass from nats_development context>
```

`Start the server`

```bash
go run server/cmd/server.go -nats "nats://localhost:<port from nats_development context>" -nats-user local -nats-pass <pass from nats_development context>
```

`Subscribe to all events`

```bash
nats sub ">" --server localhost:<port from nats_development context> --user local --password <pass from nats_development context>
```

`Publish to the service.events subject`

This is for controlling the client from the NATs cli.

```bash
nats pub "service.events" "stop:hid" --server localhost:44853 --user local --password <pass from nats_development context>
```

## Architecture

### Data Flow

The client binary is responsible for capturing HID events, screen captures, and machine metadata. It publishes these data to the NATS server under specific subjects.
