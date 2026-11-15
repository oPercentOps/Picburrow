# Getting photos from your phone to Picburrow

> **Draft, not yet tested on real phones.** The Picburrow side below is how the program works. The steps inside the
> phone apps are written from their documentation and may differ from what you see. If a step is wrong, please open an issue.

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
POST /api/assets/lookup        (signed in: POST /api/login first, same cookie)
{"photos": [{"name": "IMG_0236.HEIC", "taken": "2023-06-15 00:06:39"}, ...]}
-> {"checked": 1, "missing": [{"name": "IMG_0236.HEIC", "taken": "2023-06-15 00:06:39"}]}
```

A Shortcut that does it (untested draft, like the rest of this page; the Shortcuts app on a French iPhone has the
same actions under French names):

1. **Get Contents of URL** `http://<pi-address>:8080/api/login`, method POST, request body JSON with the text fields
   `username` and `password`. The password is then stored in the Shortcut: don't share that Shortcut.
2. **Find Photos** where *Date Taken* is in the last 30 days (or any range), up to 5000.
3. **Repeat with Each** photo:
   - **Format Date** *Date Taken*, custom format `yyyy-MM-dd HH:mm:ss`;
   - **Text** `{"name":"<Name>","taken":"<Formatted Date>"}`, with the photo's *Name* and the formatted date inserted
     as variables.
4. **Combine Text** *Repeat Results* with the custom separator `,`, then **Text** `{"photos":[<Combined Text>]}`.
5. **Get Contents of URL** `http://<pi-address>:8080/api/assets/lookup`, method POST, header
   `Content-Type: application/json`, request body *File*: the text from step 4.
6. **Get Dictionary Value** `missing`, then **Show Result** (or **Count** to just get a number).

If step 5 answers "Sign in first", the login cookie from step 1 did not carry over: please open an issue.

## Good to know

- **Deleting on the phone.** With Send Only / Receive Only, a photo deleted on the phone is also deleted on the Pi
  the next time they sync, and Picburrow moves it to Trash for 30 days (the entry, not a copy of the file). If the Pi
  should keep everything, turn on *Ignore Delete* for that folder on the Pi (an advanced folder setting in Syncthing).
- **This is not a backup of the Pi.** Back up the photo folder the way you back up anything else on that disk.
  Picburrow's own data (albums, favourites, previews) is separate and small.
- **Nothing shows up?** **Libraries → Scan now** looks at the folder immediately, and the Libraries page says so if
  the folder can't be read. **Indexing** lists files Picburrow could not make a preview of.
