//! Log tailing: last-N-lines plus polling "follow" mode.

use std::fs::{self, File};
use std::io::{self, BufRead, BufReader, Read, Seek, SeekFrom};
use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;

/// Default number of trailing lines printed before following.
pub const DEFAULT_TAIL_LINES: usize = 200;

/// How often the follow loop polls the file for new content.
const POLL_INTERVAL: Duration = Duration::from_millis(250);

/// Reads `reader` line-by-line and returns (at most) the last `n` lines,
/// oldest first, with trailing `\n`/`\r\n` stripped.
///
/// This only ever keeps `n` lines in memory at a time (via a bounded
/// `VecDeque`), regardless of how long the input is.
pub fn last_n_lines<R: Read>(reader: R, n: usize) -> io::Result<Vec<String>> {
    let mut buf = BufReader::new(reader);
    let mut window: std::collections::VecDeque<String> =
        std::collections::VecDeque::with_capacity(n);
    let mut line = String::new();
    loop {
        line.clear();
        let bytes_read = buf.read_line(&mut line)?;
        if bytes_read == 0 {
            break;
        }
        strip_newline(&mut line);
        if window.len() == n {
            window.pop_front();
        }
        if n > 0 {
            window.push_back(line.clone());
        }
    }
    Ok(window.into_iter().collect())
}

fn strip_newline(line: &mut String) {
    if line.ends_with('\n') {
        line.pop();
        if line.ends_with('\r') {
            line.pop();
        }
    }
}

/// Prints the last [`DEFAULT_TAIL_LINES`] lines of `path` as `{"line": ...}`
/// JSON lines to stdout. If `follow` is true, keeps running afterwards,
/// polling the file for appended content and emitting a JSON line per new
/// complete line, until the file disappears or stdin is closed/interrupted.
pub fn run_tail(path: &Path, follow: bool) -> io::Result<()> {
    let file = File::open(path)?;
    let mut reader = BufReader::new(file);
    let initial_lines = last_n_lines(&mut reader, DEFAULT_TAIL_LINES)?;

    let stdout = io::stdout();
    let mut out = stdout.lock();
    for line in &initial_lines {
        print_line_json(&mut out, line)?;
    }

    if !follow {
        return Ok(());
    }

    // Start following from the current end of the file.
    let mut pos = fs::metadata(path)?.len();

    let stop = Arc::new(AtomicBool::new(false));
    spawn_stdin_watcher(stop.clone());

    loop {
        if stop.load(Ordering::Relaxed) {
            return Ok(());
        }
        thread::sleep(POLL_INTERVAL);

        let metadata = match fs::metadata(path) {
            Ok(m) => m,
            Err(_) => return Ok(()), // file disappeared: exit cleanly
        };
        let len = metadata.len();

        if len < pos {
            // File was truncated or replaced (e.g. log rotation); restart
            // from the beginning of the now-shorter file.
            pos = 0;
        }
        if len == pos {
            continue;
        }

        let mut file = File::open(path)?;
        file.seek(SeekFrom::Start(pos))?;
        let mut reader = BufReader::new(file);

        loop {
            let mut buf = String::new();
            let bytes_read = reader.read_line(&mut buf)?;
            if bytes_read == 0 {
                break;
            }
            if !buf.ends_with('\n') {
                // Incomplete line at EOF: leave it unconsumed for next poll.
                break;
            }
            strip_newline(&mut buf);
            print_line_json(&mut out, &buf)?;
            pos += bytes_read as u64;
        }
    }
}

fn print_line_json<W: io::Write>(out: &mut W, line: &str) -> io::Result<()> {
    let value = serde_json::json!({ "line": line });
    writeln!(out, "{}", value)?;
    out.flush()
}

/// Spawns a background thread that blocks reading stdin to EOF, then flips
/// `stop` to true so the follow loop can exit shortly after its next poll.
fn spawn_stdin_watcher(stop: Arc<AtomicBool>) {
    thread::spawn(move || {
        let mut buf = [0u8; 64];
        loop {
            match io::stdin().read(&mut buf) {
                Ok(0) | Err(_) => {
                    stop.store(true, Ordering::Relaxed);
                    return;
                }
                Ok(_) => continue,
            }
        }
    });
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    #[test]
    fn returns_all_lines_when_fewer_than_n() {
        let data = "one\ntwo\nthree\n";
        let lines = last_n_lines(Cursor::new(data), 200).unwrap();
        assert_eq!(lines, vec!["one", "two", "three"]);
    }

    #[test]
    fn returns_only_last_n_lines() {
        let data = "1\n2\n3\n4\n5\n";
        let lines = last_n_lines(Cursor::new(data), 3).unwrap();
        assert_eq!(lines, vec!["3", "4", "5"]);
    }

    #[test]
    fn handles_no_trailing_newline_on_final_line() {
        let data = "a\nb\nc";
        let lines = last_n_lines(Cursor::new(data), 200).unwrap();
        assert_eq!(lines, vec!["a", "b", "c"]);
    }

    #[test]
    fn handles_crlf_line_endings() {
        let data = "a\r\nb\r\nc\r\n";
        let lines = last_n_lines(Cursor::new(data), 200).unwrap();
        assert_eq!(lines, vec!["a", "b", "c"]);
    }

    #[test]
    fn empty_input_yields_no_lines() {
        let lines = last_n_lines(Cursor::new(""), 200).unwrap();
        assert!(lines.is_empty());
    }

    #[test]
    fn n_zero_yields_no_lines() {
        let data = "a\nb\nc\n";
        let lines = last_n_lines(Cursor::new(data), 0).unwrap();
        assert!(lines.is_empty());
    }
}
