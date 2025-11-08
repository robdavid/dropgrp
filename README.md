# dropgrp

dropgrp runs a command with one or more supplementary groups removed from the process. It is useful when you want to execute a program without access provided by certain supplementary groups (for example, to avoid access to a shared resource).

## Synopsis

`dropgrp` &lt;group&gt;[`,`&lt;group&gt;...] &lt;command&gt; [&lt;args&gt;...]

- &lt;group&gt; — group name or numeric GIDs to drop from the calling process's supplementary groups. Multiple groups can be specified separated by commas.
- &lt;command&gt;  — the command run to run after dropping the specified groups.
- &lt;args&gt; - zero or more command arguments

Examples:
```
dropgrp docker,staff id -nG
dropgrp 1001,1002 id -nG
dropgrp docker,1001 id -nG
```
These run `id -nG` after removing the specified supplementary groups (by name or numeric GID) if they are present.

## Behavior

- The program examines the current process's supplementary groups and removes any groups that match the provided group names or numeric GIDs.
- If a provided group token parses as a number it is treated as a numeric GID. Otherwise the program looks up the group name using the system group database.
- Any requested group that is not present in the current user's supplementary groups is ignored.
- If any requested group does not exist, the program will terminate with an error.
- If any requested group is equal to the user's primary group (GID), the program will terminate with an error — only supplementary groups can be dropped.
- The `setgroups` syscall is used to modify the process' groups. This is a privileged call so if the current user is not root, the program will attempt to `seteuid` to 0 before doing so, and then immediately back to the calling user's UID. This only works if the program is setuid root.
- After adjusting groups, the program execs the requested command (replacing the current process).

## Exit codes

- `1` — incorrect usage (wrong number of arguments).
- `2` — error occurred while attempting to drop groups or exec the command. Errors and per-group warnings are printed to stderr.

## Requirements and installation

- Go toolchain to build:
  ```
  go build -o dropgrp .
  ```
- The program must be able to call `setgroups(2)`. On most systems this requires root privileges (or appropriate capabilities). Two options:
  - Run the binary as root.
  - Install the binary setuid root:
    ```
    sudo chown root:root dropgrp
    sudo chmod 4755 dropgrp
    ```
    Be careful with permissions as setting the program setuid root carries security implications (see below).
- Platform: Unix-like systems with Go `syscall.Setgroups`, `Seteuid`, and `Exec` support (Linux/Unix). Behavior may vary by OS.

## Security considerations

- Setting a binary setuid root entails risk. If the binary is writable by others or contains bugs, it may be abused to gain root privileges.
- Validate the binary's ownership and permissions before installation. Only install setuid root when you understand and accept the security implications.

## Disclaimer

No assurance is given this software is bug free. Use at your own risk.

## Notes and limitations

- The program accepts either group names or numeric GIDs as input.
- Unknown group names or invalid numeric GIDs will terminate the program with an error.
- Groups not present in the caller's supplementary list will be silently ignored.
- The program only removes groups from the supplementary group list — it does not change the primary GID or other credentials.

## Example usage

Build:
```
go build -o dropgrp .
```

Run (example using sudo to allow setgroups):
```
sudo ./dropgrp docker,1001 id -nG
```

This prints the list of groups the command runs with, which should not include `docker` (or GID 1001) if they were supplementary groups of the caller.

