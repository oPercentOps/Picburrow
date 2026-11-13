---
name: Something doesn't work
about: Report a problem with Picburrow
labels: bug
---

**What happened, and what you expected instead**


**Your setup**
- Board and memory (e.g. Raspberry Pi 3B, 1 GB):
- OS (`cat /etc/os-release | head -1`, `uname -m`):
- Picburrow version (the `.deb` file name):
- Where the photos and Picburrow's data live (SD card, USB drive, network share):

**Log** (`journalctl -u picburrow --since "1 hour ago" --no-pager | tail -50`)

```
paste here
```

**If photos are missing previews**: the list under Admin → Indexing → "Files without a preview".

Please leave out photos of people who haven't agreed to it, and remove addresses or names from paths if you prefer.
