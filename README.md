# Atella

[![License](https://img.shields.io/github/license/JIexa24/Atella)](LICENSE)

Atella. Agent for distributed checking servers status.

atella - Main daemon

```shell
Usage: ./build/atella [params]
  -config string
        Path to config
  -config-directory string
        Path to config directory
  -version
        Print version and exit
```

atella-cli - Interface for send commands to daemon

```shell
Usage: atella-cli [params]
  -channel string
        Report channel. Possible values:
                All
                Tgsibnet
                Mail
                Graphite (default "all")
  -cmd string
        Command. Possible values:
                Send
                Reload
                Rotate
                Update
                WrapConfig
                Report
  -config string
        Path to config
  -config-directory string
        Path to config directory
  -message string
        Message. Work only with run mode "Report" & report type "Custom" (default "Test")
  -print-pidfile
        Print pid file path and exit
  -to-version string
        Version for update
  -type string
        Report type. Possible values:
                Reboot
                Custom
  -version
        Print version and exit
```

