#!/bin/bash
# Builds a fake phone-backup library for e2e/slice.cjs: 128 dated photos in 8 months (landscape, portrait,
# panorama, square), 3 dateless videos, a 48 MP JPEG, a corrupt file and a Synology @eaDir that must be skipped.
# Needs ffmpeg, exiftool and vips. Usage: e2e/make-fixtures.sh <empty-dir>
set -euo pipefail
L=${1:?usage: make-fixtures.sh <empty-dir>}
mkdir -p "$L/DCIM/Camera" "$L/@eaDir"
cd "$L"
srcs=(testsrc2 mandelbrot "gradients=n=3" "cellauto=rule=110" "life=mold=10:ratio=0.3" rgbtestsrc smptehdbars "gradients=n=5:speed=0.2")
sizes=(1600x1200 1200x1600 1600x1200 1600x900 1200x1200 2400x800 1600x1200 1080x1920)
n=0
for m in 01 02 04 06 07 09 10 11; do for d in 03 11 18 27; do for k in 1 2 3 4; do
  n=$((n+1)); i=$((n % 8)); f=DCIM/Camera/IMG_2026$m${d}_$k.jpg
  ffmpeg -v error -f lavfi -i "${srcs[$i]}:size=${sizes[$i]}" -ss 0.$k -frames:v 1 -q:v 4 "$f" 2>/dev/null ||
    ffmpeg -v error -f lavfi -i "${srcs[$i]}" -vf "scale=${sizes[$i]}" -frames:v 1 -q:v 4 "$f"
  exiftool -q -overwrite_original "-DateTimeOriginal=2026:$m:$d 1$k:2$k:00" -OffsetTimeOriginal=+01:00 -Make=Google "-Model=Pixel 8" "$f"
done; done; done
for v in 1 2 3; do
  ffmpeg -v error -f lavfi -i testsrc2=size=1280x720:duration=4 -f lavfi -i sine=duration=4 -pix_fmt yuv420p -shortest "DCIM/Camera/VID_202610${v}5.mp4"
done
vips zone big.v 8000 6000 && vips jpegsave big.v DCIM/Camera/IMG_big_48mp.jpg --Q 85 && rm big.v
exiftool -q -overwrite_original "-DateTimeOriginal=2026:11:01 09:00:00" DCIM/Camera/IMG_big_48mp.jpg
echo "not really a jpeg" > DCIM/Camera/IMG_broken.jpg
cp DCIM/Camera/IMG_20260103_1.jpg @eaDir/should_be_skipped.jpg
echo "fixtures: $(find "$L" -type f | wc -l) files in $L"
