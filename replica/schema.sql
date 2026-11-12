-- RaspictureGalleryNas schema, SQLite 3.38+ for unixepoch() (WAL mode, foreign_keys=ON per connection).
-- Times: INTEGER unix seconds UTC unless noted. IDs: INTEGER rowid (smaller/faster than uuid on a Pi);
-- anything exposed publicly uses a random token instead of the id.

PRAGMA foreign_keys = ON;

create table users (
  id            integer primary key,
  username      text not null unique collate nocase,
  password_hash text not null,                       -- bcrypt
  is_admin      integer not null default 0 check (is_admin in (0,1)),
  created_at    integer not null default (unixepoch()),
  updated_at    integer not null default (unixepoch())
);

create table sessions (
  token_hash  blob primary key,                      -- sha256 of the cookie value, never the raw token
  user_id     integer not null references users(id) on delete cascade,
  expires_at  integer not null,
  created_at  integer not null default (unixepoch())
);
create index sessions_user on sessions(user_id);

create table libraries (
  id               integer primary key,
  owner_id         integer not null references users(id) on delete cascade,
  name             text not null,
  import_paths     text not null default '[]' check (json_valid(import_paths)),      -- JSON array of absolute dirs
  exclusion_globs  text not null default '["**/@eaDir/**","**/.*/**"]' check (json_valid(exclusion_globs)),
  last_scan_at     integer,
  last_scan_error  text,
  created_at       integer not null default (unixepoch()),
  updated_at       integer not null default (unixepoch())
);
create index libraries_owner on libraries(owner_id);

create table assets (
  id            integer primary key,
  library_id    integer not null references libraries(id) on delete cascade,
  path          text not null,                       -- absolute path on disk
  dir           text not null,                       -- parent dir, for folder view
  size          integer not null,
  mtime         integer not null,
  quick_hash    blob not null,                       -- sha256(size || first 64 KiB || last 64 KiB): relinks moved files
  type          text not null check (type in ('image','video')),
  mime          text not null,
  width         integer,
  height        integer,
  orientation   integer,
  duration_ms   integer,
  taken_local   text,                                -- 'YYYY-MM-DD HH:MM:SS' wall clock from EXIF; timeline sorts on this
  tz_offset_min integer,                             -- null when EXIF has no offset (common)
  camera_make   text,
  camera_model  text,
  lens          text,
  lat           real,
  lon           real,
  city          text,
  country       text,
  description   text,
  rating        integer check (rating between 0 and 5),
  live_video_id integer references assets(id) on delete set null,  -- still -> its motion clip
  is_favorite   integer not null default 0 check (is_favorite in (0,1)),
  is_archived   integer not null default 0 check (is_archived in (0,1)),
  thumbhash     blob,                                -- blur placeholder, ~25 bytes
  thumbs_at     integer,                             -- null = small/large thumbnails not generated yet
  thumb_error   text,                                -- unreadable/unsupported file; shown, not retried forever
  missing_at    integer,                             -- file gone on rescan = in trash; purged after 30 days
  created_at    integer not null default (unixepoch()),
  updated_at    integer not null default (unixepoch()),
  unique (library_id, path)
);
create index assets_timeline on assets(library_id, taken_local desc) where missing_at is null;
create index assets_dir      on assets(library_id, dir);
create index assets_hash     on assets(quick_hash);
create index assets_todo     on assets(id) where thumbs_at is null and thumb_error is null and missing_at is null;
create index assets_live     on assets(live_video_id);
create index assets_missing  on assets(missing_at) where missing_at is not null;

create table albums (
  id             integer primary key,
  owner_id       integer not null references users(id) on delete cascade,
  name           text not null,
  cover_asset_id integer references assets(id) on delete set null,
  created_at     integer not null default (unixepoch()),
  updated_at     integer not null default (unixepoch())
);
create index albums_owner on albums(owner_id);

create table album_assets (
  album_id  integer not null references albums(id) on delete cascade,
  asset_id  integer not null references assets(id) on delete cascade,
  added_at  integer not null default (unixepoch()),
  primary key (album_id, asset_id)
) without rowid;
create index album_assets_asset on album_assets(asset_id);

create table shared_links (
  id             integer primary key,
  album_id       integer not null references albums(id) on delete cascade,
  token          text not null unique,               -- >= 128 bits random, base64url
  password_hash  text,                               -- bcrypt, null = no password
  allow_download integer not null default 1 check (allow_download in (0,1)),
  expires_at     integer,
  created_at     integer not null default (unixepoch())
);
create index shared_links_album on shared_links(album_id);

create table tags (
  id         integer primary key,
  owner_id   integer not null references users(id) on delete cascade,
  parent_id  integer references tags(id) on delete cascade,
  name       text not null
);
create unique index tags_unique on tags(owner_id, coalesce(parent_id, 0), name);  -- NULL parents aren't equal in a plain unique
create index tags_parent on tags(parent_id);

create table asset_tags (
  asset_id integer not null references assets(id) on delete cascade,
  tag_id   integer not null references tags(id) on delete cascade,
  primary key (asset_id, tag_id)
) without rowid;
create index asset_tags_tag on asset_tags(tag_id);
