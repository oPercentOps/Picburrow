# Getting photos from your phone to Picburrow

> **The iPhone Shortcuts (check and catch-up) are tested on a real iPhone, at home and from outside.** The steps inside
> Syncthing, photo-transfer apps and NAS apps are written from their documentation and may differ from what you see.
> If a step is wrong, please open an issue.

Picburrow does not copy photos off your phone. It shows the photos it finds in a folder, and it never changes them.
So the job has two halves:

1. something copies new photos from the phone into a folder the Pi can read;
2. that folder is one of Picburrow's libraries.

New photos show up in Picburrow within about 15 minutes of landing in the folder.

## Pick a way to copy

| | Phone | What you install | Works in the background |
| --- | --- | --- | --- |
| **1. Syncthing** | Android | Syncthing on the Pi, a Syncthing app on the phone | yes |
| **2. A photo-transfer app** | iPhone (and Android) | an app that sends new photos to a shared folder on the Pi or NAS | now and then: iOS decides when |
| **3. Your NAS's own app** | both | nothing new, if your NAS already backs up phones | depends on the app |

If phone backups already land on a NAS or disk the Pi can read, skip to [Tell Picburrow about the folder](#tell-picburrow-about-the-folder).

## Option 1: Syncthing (Android)

Syncthing keeps a folder on the phone and a folder on the Pi the same, over your home network, with no account anywhere.

**On the Pi**

```bash
sudo apt install syncthing
sudo systemctl enable --now syncthing@$USER
sudo mkdir -p /srv/photos && sudo chown $USER /srv/photos   # where the phones' folders will go
```

Its settings page is at `http://127.0.0.1:8384` on the Pi. From another computer, the simplest safe way to open it is
an SSH tunnel: `ssh -L 8384:127.0.0.1:8384 <you>@<pi-address>`, then open `http://127.0.0.1:8384` on that computer.

**On the phone**

1. Install a Syncthing app for Android and open it.
2. Add the Pi as a device: on the Pi's settings page choose *Actions → Show ID* and scan the code with the phone.
   Accept the phone on the Pi's page when it asks.
3. Share the phone's camera folder (usually `DCIM/Camera`) with the Pi. Set the folder type on the phone to
   **Send Only**, so nothing on the Pi can change or delete photos on the phone.

**Back on the Pi**

4. Accept the folder when the Pi's page offers it. Choose where it goes, for example `/srv/photos/ann-phone`,
   and set its type to **Receive Only**.

One phone, one folder. Do the same for each family member, each with their own folder.

## Option 2: a photo-transfer app (iPhone)

An iPhone will not let an app run all the time, so no app can copy every photo the moment it is taken. Apps in this
group copy new photos when iOS gives them a turn (often while charging on Wi-Fi) or when you open them.

1. Share a folder from the Pi or NAS on your home network (SMB is the usual way; on a NAS it is a "shared folder").
2. Install a photo-transfer app that can send to an SMB share, and point it at that folder.
3. Turn on its automatic transfer of new photos. Open the app once in a while if photos lag behind.

Keep the photos in their original format (HEIC and live photos included). Picburrow shows them as they are.

## Option 3: your NAS's own app

If your NAS vendor's phone app already backs up photos to the NAS, keep it. Mount that NAS folder on the Pi
(read-only is fine and safest) and use it as the library folder.

## Tell Picburrow about the folder

1. Check that Picburrow can read it: `sudo -u picburrow ls /srv/photos/ann-phone`.
   If that fails, the [README](../README.md#install-on-a-raspberry-pi-64-bit-raspberry-pi-os) explains folder permissions,
   and the Libraries page names the folder that blocks the way.
2. In Picburrow:
   - your own phone: **Libraries → add the folder** to your library;
   - a family member's phone: **People → Add someone**, with that folder as theirs. To let them see the family
     library too, press **Share my photos** next to their name.

## Check that nothing is missing (iPhone Shortcut)

Picburrow can tell which photos on the phone are not in its index yet: `POST /api/assets/lookup` takes the name and
the time taken of up to 5000 photos and answers with the ones it can't find. A photo counts as found when one in your
libraries has the same time taken to the second, or has its name (`IMG_0236`, whatever the extension or the prefix
the copy tool added) in its file name, taken within a day either side. Archived photos count as found; photos in
Trash do not.

```
POST /api/assets/lookup        Authorization: Bearer <API key>   (or signed in with the app's cookie)
{"photos": [{"name": "IMG_0236.HEIC", "taken": "2023-06-15 00:06:39"}, ...]}
  or, one photo per line, in the same order:
{"names": "IMG_0236.HEIC\nIMG_0237.HEIC", "dates": "2023-06-15T00:06:39+02:00\n2023-06-15T00:07:02+02:00"}
-> {"checked": 2, "missing": [{"name": "IMG_0236.HEIC", "taken": "2023-06-15T00:06:39+02:00"}]}
  or file names alone, one per line, as your copy tool names the files:
{"names": "2023-06-15_000639_IMG_0236\n2023-06-15_000702_IMG_0237"}
-> {"checked": 2, "missing": [{"name": "2023-06-15_000702_IMG_0237"}]}
```

Dates are `2023-06-15 00:06:39` (the clock time on the photo) or ISO 8601; an ISO date is read in the server's time
zone. The second form is the one to use from a Shortcut: it needs no loop over each photo, which on an iPhone takes
around half a second per photo.

With file names alone, no date is compared: a photo is there if a file in your libraries has exactly that name,
whatever its extension (`.HEIC`, `.jpeg`), letter case or duplicate suffix (`-2`). Use it when the copy tool puts the
date in the name: it also finds photos whose file has no date inside (screenshots, saved images), which the other
forms can miss when the file was copied more than a day after it was taken.

**The key.** In Picburrow, **API keys → Make a key** (each person makes their own; on a phone it is under **More**).
Copy it then: it is shown only once. A key opens this lookup and the upload below, nothing else, so a Shortcut
holding it can't show, change, delete or download photos. Delete it on the same page and it stops working at once; the page also shows when each
key was last used.

A Shortcut that does it, a thousand photos per request (the Shortcuts
app on a French iPhone has the same actions under French names):

1. **Ask for Input**, a number of days, no default; **If** it has no value, use 36500 (the whole library).
2. **Find Photos** where *Date Taken* is in those last days, sorted by *Date Taken*, oldest first, no limit.
   **Count** them (*Total*), then **Calculate** *Total* / 1000 and **Round Number** up (*Batches*).
3. **Repeat** *Batches* times:
   - **Calculate** *Start* = (*Repeat Index* − 1) × 1000 + 1 and *End* = the smaller of *Repeat Index* × 1000 and *Total*;
   - **Get Items from List**, *Items in Range* *Start* to *End*;
   - **Get Details of Photo Media**: *Name*, then **Combine Text** with *New Lines*;
   - **Get Details of Photo Media**: *Date Taken*, then **Combine Text** with *New Lines*. Tap the *Date Taken*
     variable and set its date format to **ISO 8601**, with the time included;
   - **Get Contents of URL** `http://<pi-address>:8080/api/assets/lookup`, method POST, headers
     `Authorization: Bearer <your key>`, request body *JSON* with two *Text* fields: `names` (the combined names) and
     `dates` (the combined dates). Shortcuts writes the JSON itself, quotes and line breaks included;
   - **Get Dictionary Value** `missing`, then **Combine Text** with *New Lines* and add it to a list. Don't ask
     for `name` on that list: *Get Dictionary Value* takes one dictionary, so Shortcuts would ask you to pick one.
4. **Save File**: the list as text, for example to *Shortcuts/Picburrow-missing.txt*, replacing the old one.

Over the internet, use the HTTPS address from [remote-access](remote-access.md): over plain HTTP the key travels
readable.

## Send the missing photos (catch-up, from anywhere)

When the phone can't reach the NAS share (away from home, or no SMB app), the same key sends the photos straight to
Picburrow, over the Tailscale address from [remote-access](remote-access.md):

```
POST /api/assets/upload        Authorization: Bearer <API key>   (the key is required: the login cookie isn't enough)
multipart/form-data, one or more files in the field file
-> {"saved": [{"name": "2023-06/IMG_0236.HEIC"}, {"name": "2023-06/IMG_0237-2.HEIC"}, {"name": "2023-07/IMG_0238.HEIC", "existed": true}]}
```

```bash
curl -H "Authorization: Bearer <key>" -F file=@IMG_0236.HEIC https://<pi>.<tailnet>.ts.net/api/assets/upload
```

- The photo lands in the **first folder of your own library** (Libraries page), in a subfolder for the month it was
  taken (`2023-06`, made if missing), under the name the phone gives it, and shows up once the scan that starts right
  after has read it. The month comes from the date inside the file; without one, from a name that starts with it
  (`2023-06-15_000639_IMG_0236`, as the Shortcut below names them); otherwise it is the month of the upload.
- Nothing is ever overwritten. A file of the same name and the same size in that month's folder counts as the same photo (`existed`, nothing
  written), so sending a batch again after a dropped connection is harmless; a different file of the same name is
  saved as `IMG_0236-2.HEIC`, which the lookup by file name also counts as found.
- Only photo and video files Picburrow shows are taken. It refuses when less than 1 GB would be left on that disk.
- **The picburrow user must be allowed to write to that folder** (until now it only needed to read it):
  `sudo -u picburrow touch /srv/photos/ann-phone/.test && sudo rm /srv/photos/ann-phone/.test` tells you. For a
  folder under `/home`, the service also keeps `/home` read-only: `sudo systemctl edit picburrow`, add
  `[Service]` and `ReadWritePaths=/home/<you>/Pictures`, then `sudo systemctl restart picburrow`.
- **Syncthing *Receive Only* folder:** Syncthing lists the uploaded photos as *local additions*; don't press
  *Revert Local Changes*, it would delete them. Send Only from the phone keeps working as before.

A Shortcut that does the whole catch-up, tested on an iPhone (with the lookup by file name, the copy tool naming the
files `2023-06-15_000639_IMG_0236.HEIC`; up to 5000 photos, so keep the number of days small):

1. **Ask for Input**, a number of days; **If** it has no value, use 36500.
2. **Find Photos** where *Creation Date* is after today minus those days, *Is a Screenshot* is off, oldest first.
3. **Repeat with Each** photo: **Get Details of Images** *Creation Date* and *Name*, then **Text**
   `<Creation Date>_<Name>`, the date in the custom format `yyyy-MM-dd_HHmmss`. After the loop, **Combine Text**
   with *New Lines*.
4. **Get Contents of URL** `https://<pi>.<tailnet>.ts.net/api/assets/lookup`, method POST, header
   `Authorization: Bearer <your key>`, request body *JSON* with one *Text* field `names` = the combined text.
5. **Get Dictionary Value** `missing`, **Count** it, and **If** the count is more than 0:
   - **Combine Text** the `missing` value itself (the whole list of `{"name": ...}`) into a variable *Missing*;
   - **Repeat with Each** photo of step 2:
     - **Get Item from List**: the photos of step 2, *Item at Index* *Repeat Index*;
     - **Get Details of Images** of that item: *Creation Date*, *Name*, *File Extension*, and the same **Text**
       `<Creation Date>_<Name>` as in step 3;
     - **If** *Missing* contains that text: **Set Name** of the item to `<Text>.<File Extension>`, then
       **Get Contents of URL** `https://<pi>.<tailnet>.ts.net/api/assets/upload`, method POST, the same
       `Authorization` header, request body *Form* with a field `file` of type *File* = the renamed item.
   - **Show Notification** with the count; **Otherwise**, a notification that nothing is missing.

Renaming matters: the photo is saved under the name it is sent with, and the next lookup must find
`2023-06-15_000639_IMG_0236` again, or it would send the photo every time. Shortcuts may send a JPEG instead of the
HEIC original; Picburrow takes either.

## Good to know

- **Deleting on the phone.** With Send Only / Receive Only, a photo deleted on the phone is also deleted on the Pi
  the next time they sync, and Picburrow moves it to Trash for 30 days (the entry, not a copy of the file). If the Pi
  should keep everything, turn on *Ignore Delete* for that folder on the Pi (an advanced folder setting in Syncthing).
- **This is not a backup of the Pi.** Back up the photo folder the way you back up anything else on that disk.
  Picburrow's own data (albums, favourites, previews) is separate and small.
- **Nothing shows up?** **Libraries → Scan now** looks at the folder immediately, and the Libraries page says so if
  the folder can't be read. **Indexing** lists files Picburrow could not make a preview of.
