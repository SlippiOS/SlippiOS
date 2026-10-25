//! Mapping GameCube pad state to the axes and buttons of a virtual evdev pad.

use crate::protocol::PadState;

pub const AXIS_MAX: i16 = i16::MAX;

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct GamepadState {
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
    pub left_x: i16,
    pub left_y: i16,
    pub right_x: i16,
    pub right_y: i16,
    pub trigger_l: i16,
    pub trigger_r: i16,
}

/// Convert the adapter's unsigned stick range to a centered signed axis.
pub fn axis(value: u8) -> i16 {
    ((value as i32 - 128) * 258).clamp(i16::MIN as i32, i16::MAX as i32) as i16
}

/// GameCube Y axes grow downward; Linux expects up to be positive.
pub fn axis_y(value: u8) -> i16 {
    (-(axis(value) as i32)).clamp(i16::MIN as i32, i16::MAX as i32) as i16
}

/// Triggers are 0..255 on the adapter and 0..32767 on the virtual pad.
pub fn trigger(value: u8) -> i16 {
    (value as i32 * 128).min(AXIS_MAX as i32) as i16
}

impl From<PadState> for GamepadState {
    fn from(pad: PadState) -> Self {
        Self {
            a: pad.a,
            b: pad.b,
            x: pad.x,
            y: pad.y,
            start: pad.start,
            z: pad.z,
            l: pad.l,
            r: pad.r,
            dpad_left: pad.dpad_left,
            dpad_right: pad.dpad_right,
            dpad_down: pad.dpad_down,
            dpad_up: pad.dpad_up,
            left_x: axis(pad.stick_x),
            left_y: axis_y(pad.stick_y),
            right_x: axis(pad.c_stick_x),
            right_y: axis_y(pad.c_stick_y),
            trigger_l: trigger(pad.trigger_l),
            trigger_r: trigger(pad.trigger_r),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn centers_sticks() {
        assert_eq!(axis(0), -32768);
        assert_eq!(axis(128), 0);
        assert_eq!(axis(255), 32766);
        assert_eq!(axis_y(0), 32767);
        assert_eq!(axis_y(128), 0);
        assert_eq!(axis_y(255), -32766);
    }

    #[test]
    fn clamps_axes() {
        assert!(axis(0) >= -32768);
        assert_eq!(axis(255), 32766);
        assert_eq!(axis_y(0), 32767);
    }

    #[test]
    fn scales_triggers() {
        assert_eq!(trigger(0), 0);
        assert_eq!(trigger(255), 32640);
    }

    #[test]
    fn maps_buttons_and_axes() {
        let pad = PadState {
            connected: true,
            a: true,
            z: true,
            dpad_up: true,
            stick_x: 255,
            stick_y: 0,
            c_stick_x: 128,
            c_stick_y: 128,
            trigger_l: 255,
            trigger_r: 0,
            ..PadState::default()
        };
        let mapped = GamepadState::from(pad);
        assert!(mapped.a && mapped.z && mapped.dpad_up);
        assert_eq!(mapped.left_x, 32766);
        assert_eq!(mapped.left_y, 32767);
        assert_eq!(mapped.right_x, 0);
        assert_eq!(mapped.right_y, 0);
        assert_eq!(mapped.trigger_l, 32640);
        assert_eq!(mapped.trigger_r, 0);
    }
}
