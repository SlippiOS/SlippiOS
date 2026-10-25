//! Parsing for the Wii U GameCube adapter's interrupt input report.

pub const REPORT_ID: u8 = 0x21;
pub const PORTS: usize = 4;
pub const PORT_BYTES: usize = 9;
pub const REPORT_LEN: usize = 1 + PORTS * PORT_BYTES;

const STATUS_WIRED: u8 = 0x10;
const STATUS_WIRELESS: u8 = 0x20;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ParseError {
    WrongLength(usize),
    WrongReport(u8),
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PadState {
    pub connected: bool,
    pub wireless: bool,
    pub a: bool,
    pub b: bool,
    pub x: bool,
    pub y: bool,
    pub start: bool,
    pub z: bool,
    pub l: bool,
    pub r: bool,
    pub dpad_left: bool,
    pub dpad_right: bool,
    pub dpad_down: bool,
    pub dpad_up: bool,
    pub stick_x: u8,
    pub stick_y: u8,
    pub c_stick_x: u8,
    pub c_stick_y: u8,
    pub trigger_l: u8,
    pub trigger_r: u8,
}

/// Parse one 37-byte interrupt report into four controller states.
pub fn parse(report: &[u8]) -> Result<[PadState; PORTS], ParseError> {
    if report.len() != REPORT_LEN {
        return Err(ParseError::WrongLength(report.len()));
    }
    if report[0] != REPORT_ID {
        return Err(ParseError::WrongReport(report[0]));
    }

    let mut pads = [PadState::default(); PORTS];
    for (index, pad) in pads.iter_mut().enumerate() {
        let offset = 1 + index * PORT_BYTES;
        let status = report[offset];
        let buttons0 = report[offset + 1];
        let buttons1 = report[offset + 2];

        *pad = PadState {
            connected: status & (STATUS_WIRED | STATUS_WIRELESS) != 0,
            wireless: status & STATUS_WIRELESS != 0,
            a: buttons0 & 0x01 != 0,
            b: buttons0 & 0x02 != 0,
            x: buttons0 & 0x04 != 0,
            y: buttons0 & 0x08 != 0,
            start: buttons0 & 0x10 != 0,
            dpad_left: buttons1 & 0x01 != 0,
            dpad_right: buttons1 & 0x02 != 0,
            dpad_down: buttons1 & 0x04 != 0,
            dpad_up: buttons1 & 0x08 != 0,
            z: buttons1 & 0x10 != 0,
            r: buttons1 & 0x20 != 0,
            l: buttons1 & 0x40 != 0,
            stick_x: report[offset + 3],
            stick_y: report[offset + 4],
            c_stick_x: report[offset + 5],
            c_stick_y: report[offset + 6],
            trigger_l: report[offset + 7],
            trigger_r: report[offset + 8],
        };
    }
    Ok(pads)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn report() -> [u8; REPORT_LEN] {
        let mut report = [0u8; REPORT_LEN];
        report[0] = REPORT_ID;
        report
    }

    #[test]
    fn rejects_bad_reports() {
        assert_eq!(parse(&[]), Err(ParseError::WrongLength(0)));
        let short = [REPORT_ID; REPORT_LEN - 1];
        assert_eq!(parse(&short), Err(ParseError::WrongLength(REPORT_LEN - 1)));
        let mut wrong = report();
        wrong[0] = 0x11;
        assert_eq!(parse(&wrong), Err(ParseError::WrongReport(0x11)));
    }

    #[test]
    fn parses_all_ports() {
        let mut report = report();
        for port in 0..PORTS {
            let offset = 1 + port * PORT_BYTES;
            report[offset] = STATUS_WIRED;
            report[offset + 3] = (port as u8 + 1) * 10;
            report[offset + 4] = (port as u8 + 1) * 20;
            report[offset + 5] = (port as u8 + 1) * 30;
            report[offset + 6] = (port as u8 + 1) * 40;
            report[offset + 7] = (port as u8 + 1) * 50;
            report[offset + 8] = (port as u8 + 1) * 60;
        }

        let pads = parse(&report).unwrap();
        for (port, pad) in pads.iter().enumerate() {
            let scale = port as u8 + 1;
            assert!(pad.connected);
            assert!(!pad.wireless);
            assert_eq!(pad.stick_x, scale * 10);
            assert_eq!(pad.stick_y, scale * 20);
            assert_eq!(pad.c_stick_x, scale * 30);
            assert_eq!(pad.c_stick_y, scale * 40);
            assert_eq!(pad.trigger_l, scale * 50);
            assert_eq!(pad.trigger_r, scale * 60);
        }
    }

    #[test]
    fn parses_buttons() {
        let mut report = report();
        report[1] = STATUS_WIRED;
        report[2] = 0x1f; // A B X Y Start
        report[3] = 0x7f; // left right down up Z R L

        let pads = parse(&report).unwrap();
        let pad = pads[0];
        assert!(pad.a && pad.b && pad.x && pad.y && pad.start);
        assert!(pad.dpad_left && pad.dpad_right && pad.dpad_down && pad.dpad_up);
        assert!(pad.z && pad.r && pad.l);
    }

    #[test]
    fn distinguishes_empty_and_wireless_ports() {
        let mut report = report();
        report[1] = 0x00;
        report[1 + PORT_BYTES] = STATUS_WIRED;
        report[1 + 2 * PORT_BYTES] = STATUS_WIRELESS;
        report[1 + 3 * PORT_BYTES] = 0x22;

        let pads = parse(&report).unwrap();
        assert!(!pads[0].connected);
        assert!(pads[1].connected && !pads[1].wireless);
        assert!(pads[2].connected && pads[2].wireless);
        assert!(pads[3].connected && pads[3].wireless);
    }
}
