# Local tools folder

Drop optional helper binaries here if you do not want to install them globally.

Supported lookup locations for `amdl`:

1. normal `PATH`
2. workspace root beside `amdl.exe`
3. this `tools/` folder

Examples:

- `tools/mp4decrypt.exe`
- `tools/ffmpeg.exe`
- `tools/MP4Box.exe`

## Music Video prerequisites

For MV downloads to work, you still need both:

- `mp4decrypt.exe`
- a valid `media-user-token`

`amdl doctor` will keep warning until both are available.
