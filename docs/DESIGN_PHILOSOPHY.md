# drivectl Design Philosophy

## CLI Command Structure: Generic vs. Native

A core design principle for `drivectl` is finding the right balance between the generic, dynamic `call` command and specific, native subcommands (like `get`, `list`, and `upload`).

### The `call` Command
The `call` command is designed to be a lightweight, dynamic bridge to the Google API Discovery Document. It allows users to quickly invoke any API endpoint by simply providing the service method and a JSON payload.

**Constraints of `call`:**
- **Keep it uncomplicated:** It is strictly meant for standard REST/JSON interactions.
- **No Complex Protocols:** We avoid adding complex protocol handling (like multipart/related media uploads or multi-step resumable uploads) to the `call` command. Adding these would bloat the command's generic nature, require confusing flags (e.g., `--media-file`), and make the CLI harder to use.

### Native Subcommands
For any operation that requires a complex protocol, binary data streaming, or significant user-experience optimizations, we prefer to create a **native subcommand** (e.g., `drivectl upload <file>`).

**Benefits of Native Commands:**
- **Ergonomics:** `drivectl upload video.mp4` is much easier to type and understand than a generic call with complex flags.
- **Robustness:** We can leverage the strongly typed Google Client Libraries (like the Go SDK for Google Drive) to handle the nuances of media uploads, retries, and formatting.
- **User Experience:** Native commands can provide targeted success messages, progress bars, and custom flag handling.

### Conclusion
If an API interaction can be accomplished with a simple JSON payload, it belongs in `call`. If it requires media streaming or complex protocols, it warrants a dedicated, native subcommand.