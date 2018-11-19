# Raspberry Pi IR Remote Control

Goals:
1. Using a Raspberry Pi and an IR LED, send IR codes to control audio/video
   equipment.
2. Control the Raspberry Pi via voice commands through Google Home.

There are a number of projects like this around the internet. The hardest
part for me – and for others, it seems – was the IR driver/lib configuration,
so this project documents what worked for me given my combination of hardware
and software.

## Send IR codes with IR LED

My setup:
- Raspberry Pi 3 Model B
- Raspbian Stretch Lite, release date 2017-09-07

### Step 0: Set up the hardware

There are many ways to set up your LED driver circuit. I used a basic transistor
circuit like the one depicted at http://www.raspberry-pi-geek.com/Archive/2015/10/Raspberry-Pi-IR-remote.

Make note of the GPIO pin connected to the base of the transistor, as that's
the pin we need to control to drive the LED. In my case that's pin 23 (you'll
see this number in the configuration instructions below).

**TIP:** Your eyes can't see infrared light, but your phone's camera can.
Useful for debugging your circuit.

### Step 1: Install [LIRC](http://www.lirc.org/)

    sudo apt-get install lirc

**NOTE:** I'm using stretch (Debian 9), so this installs LIRC 0.9.4c. 0.9.4 is
quite different from 0.9.0, which is what you'll get if you're running jessie
(Debian 8). If you have 0.9.0 Step 3 will be different<sup>1</sup>.

### Step 2: Enable `lirc-rpi` kernel module

We're going to do this via the device tree by editing `/boot/config.txt`.

    sudo vim /boot/config.txt

You'll see these lines in `config.txt`:

    # Uncomment this to enable the lirc-rpi module
    #dtoverlay=lirc-rpi

Change them to look like this:

    # Uncomment this to enable the lirc-rpi module
    dtoverlay=lirc-rpi,gpio_in_pin=22,gpio_out_pin=23

See how `gpio_out_pin` is set to 23? If you're not using pin 23 change that.
You can ignore `gpio_in_pin`. It's used for an IR receiver. TODO(mtraver)
document that if I ever actually use the receiver for anything.

Optional: To enable more verbose logging (which you'll find in `dmesg`), add
`debug=1` like this:

    # Uncomment this to enable the lirc-rpi module
    dtoverlay=lirc-rpi,gpio_in_pin=22,gpio_out_pin=23,debug=1

**DO NOT** edit `/etc/modules`. Other tutorials may mention putting something
similar to what we added to `config.txt` into `/etc/modules`. This is
unnecessary.

**DO NOT** add a file in `/etc/modprobe.d`. Other tutorials may mention putting
something similar to what we added to `config.txt` into a file like
`/etc/modprobe.d/ir-remote.conf` or `/etc/modprobe.d/lirc.conf`. This is
unnecessary.

### Step 3: Configure LIRC

The default LIRC configuration does not enable transmitting. From the LIRC
[configuration guide](http://www.lirc.org/html/configuration-guide.html):

> From 0.9.4+ LIRC is distributed with a default configuration based on
> the devinput driver. This should work out of the box with the following
> limitations:
>
> - There must be exactly one capture device supported by the kernel
> - The remote(s) used must be supported by the kernel.
> - There is no need to do IR blasting (i. e., to send IR data).

Let's fix that.

    sudo vim /etc/lirc/lirc_options.conf

Change `driver` to `default` and `device` to `/dev/lirc0`. Here's the diff
between the default config and my config:

    $ diff -u3 /etc/lirc/lirc_options.conf.dist /etc/lirc/lirc_options.conf
    --- /etc/lirc/lirc_options.conf.dist  2017-04-05 20:23:20.000000000 -0700
    +++ /etc/lirc/lirc_options.conf 2017-10-14 14:58:18.584886645 -0700
    @@ -8,8 +8,8 @@

     [lircd]
     nodaemon        = False
    -driver          = devinput
    -device          = auto
    +driver          = default
    +device          = /dev/lirc0
     output          = /var/run/lirc/lircd
     pidfile         = /var/run/lirc/lircd.pid
     plugindir       = /usr/lib/arm-linux-gnueabihf/lirc/plugins


**DO NOT** edit or add `hardware.conf`. Other tutorials may mention making
changes to `/etc/lirc/hardware.conf`. LIRC 0.9.4 does not use
`hardware.conf`<sup>2</sup>.

### Step 4: Add remote control config files

We need to tell LIRC which codes to transmit to talk to the equipment we wish
to control. LIRC maintains a repo of config files for many remote controls:
https://sourceforge.net/projects/lirc-remotes/

Find the one for your remote control and place it in `/etc/lirc/lircd.conf.d`.
As long as it has a `.conf` extension it'll be picked up.

If there isn't an existing config for your remote, you're in for an adventure...
I happen to be controlling a Cambridge Audio CXA60 amp with my Raspberry Pi and
there was no config file for it so I made one. It's checked into this repo.

Here's what my config directory looks like:

    $ ll /etc/lirc/lircd.conf.d
    total 52
    drwxr-xr-x 2 root root  4096 Oct 14 11:49 .
    drwxr-xr-x 3 root root  4096 Oct 14 14:58 ..
    -rw-r--r-- 1 root root  2679 Oct 14 11:49 cxa_cxc_cxn.lircd.conf
    -rw-r--r-- 1 root root 33704 Apr  5  2017 devinput.lircd.conf
    -rw-r--r-- 1 root root   615 Apr  5  2017 README.conf.d

### Step 5: Reboot

    sudo reboot

Some sanity checks for modules and services and stuff after you reboot:

    $ dmesg | grep lirc
    [    3.276240] lirc_dev: IR Remote Control driver registered, major 243
    [    3.285866] lirc_rpi: module is from the staging directory, the quality is unknown, you have been warned.
    [    4.340562] lirc_rpi: auto-detected active low receiver on GPIO pin 22
    [    4.340882] lirc_rpi lirc_rpi: lirc_dev: driver lirc_rpi registered at minor = 0
    [    4.340888] lirc_rpi: driver registered!
    [   11.929858] input: lircd-uinput as /devices/virtual/input/input0

    $ lsmod | grep lirc
    lirc_rpi                9032  3
    lirc_dev               10583  1 lirc_rpi
    rc_core                24377  1 lirc_dev

    $ ll /dev/lirc0
    crw-rw---- 1 root video 243, 0 Oct 14 17:08 /dev/lirc0

    $ ps aux | grep lirc
    root       343  0.0  0.1   4208  1084 ?        Ss   17:08   0:00 /usr/bin/irexec /etc/lirc/irexec.lircrc
    root       381  0.0  0.1   4280  1140 ?        Ss   17:08   0:00 /usr/sbin/lircmd --nodaemon
    root       516  0.4  0.4   7316  3980 ?        Ss   17:09   0:00 /usr/sbin/lircd --nodaemon
    root       517  0.0  0.1   4284  1164 ?        Ss   17:09   0:00 /usr/sbin/lircd-uinput
    pi         574  0.0  0.0   4372   552 pts/0    S+   17:09   0:00 grep --color=auto lirc

### Step 6: Test

    irsend SEND_ONCE cambridge_cxa KEY_POWER_ON

Replace `cambridge_cxa` with the contents of the `name` field from your remote
control config file, and `KEY_POWER_ON` with some code from the `codes` section.

At the very least this should execute without errors. If you enabled debugging
in the device tree (see Step 2) you can get some insight into what happened
by executing `dmesg | grep lirc`. Use your phone camera to watch the LED light
up.

## Control via voice commands

**NOTE:** I built this before Google launched [smart home Actions](https://developers.google.com/actions/smarthome/).
At the time there were only conversation-based Actions, which don't fit this use
case. A smart home Action is a better solution than what I describe below.

I wanted to issue IR codes by voice, so I did the following:
- Wrote a web server that runs on the Raspberry Pi. It exposes one endpoint for
  each IR code (e.g. /volup to turn up the volume), and `POST`ing to the
  endpoint will call the `irsend` command line utility to issue the code. See
  below for more info on deploying the web server.
- Exposed the web server to the internet using [ngrok](https://ngrok.com/).
  The docs are great so I leave this step as an exercise for the reader.
- Use [IFTTT webhooks](https://ifttt.com/maker_webhooks) to set up rules such
  that when I say something like "Ok Google, it's music time" to my Google Home,
  IFTTT fires off a request to the ngrok endpoint that points to the Raspberry
  Pi, instructing it to issue the IR code that turns on the sound system.

### Build the server

The server is written in Go. The main package is `cmd/server/main.go`. This repo
includes a Makefile that builds it for your host architecture as well as ARMv6
(e.g. Raspberry Pi Zero W) and ARMv7 (e.g. Raspberry Pi 3 B<sup>3</sup>). It
will produce binaries in the `out` directory.

### Security!

**NOTE:** Again, I built this before Google launched
[smart home Actions](https://developers.google.com/actions/smarthome/). Using
smart home Actions is the most secure and elegant way to do this.

Security is good. The knobs available to us aren't great [insert rant here about
the current state of IoT security] but we'll do what we can to lock it down.

- ngrok ([config options here](https://ngrok.com/docs#config))
    - ngrok can do HTTP basic auth. Use it. Of course the password is in the
      clear in your IFTTT rule but it's better than nothing.
    - Set `bind_tls: true` in your config to expose only an HTTPS endpoint.
    - Set `inspect: false` in your config to disable request inspection.
- Web server
    - The web server has a basic token check built in. In the JSON payload
      `POST`ed by IFTTT, include a `token` key. If its value doesn't match
      the token defined on the Raspberry Pi it will stop and return a 403.

That's all that's possible as far as I can tell. IFTTT webhooks don't allow for
any kind of secure token authentication.

### Running the server

The web server is a statically linked binary. We'll use systemd to start it up
and keep it running.

1. Place the `server` binary built for your required architecture in `/home/pi`.
2. This repo contains a systemd service definition at `config/systemd/irremote.service`.
   Copy it into the `/lib/systemd/system` directory on the Raspberry Pi.
3. To enable and start the service, run

   ```
   sudo systemctl enable irremote.service
   sudo systemctl start irremote.service
   ```

## Footnotes
[1] Other projects around the internet tend to be built using 0.9.0, leading to
some frustration while configuring, even though configuring 0.9.4 is a nicer
experience. I hope this project can help others in the same boat!

[2] Alec Leamas, LIRC maintainer, states
[here](http://lirc.10951.n7.nabble.com/Re-lirc-installation-on-raspberry-pi-running-Raspbian-jessie-tp10721p10725.html)
that "0.9.4 does not use hardware.conf."

[3] "How can this be!? The Raspberry Pi 3 B uses the BCM2837, a 64-bit ARMv8
SoC!" you exclaim. "That is correct," I reply, "but Raspbian is 32-bit only so
the chip runs in 32-bit mode. It therefore cannot execute ARMv8 binaries."
