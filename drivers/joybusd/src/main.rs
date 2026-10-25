use std::env;
use std::process::ExitCode;

use joybusd::protocol;

fn decode(hex: &str) -> Result<(), String> {
    let digits: String = hex
        .chars()
        .filter(|character| !character.is_whitespace())
        .collect();
    if digits.len() % 2 != 0 {
        return Err("hex input has an odd number of digits".into());
    }
    let bytes = (0..digits.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&digits[index..index + 2], 16))
        .collect::<Result<Vec<_>, _>>()
        .map_err(|error| format!("invalid hex: {error}"))?;
    let pads = protocol::parse(&bytes).map_err(|error| format!("{error:?}"))?;
    for (index, pad) in pads.iter().enumerate() {
        println!("port {}: {pad:?}", index + 1);
    }
    Ok(())
}

fn main() -> ExitCode {
    let arguments: Vec<String> = env::args().skip(1).collect();
    match arguments.as_slice() {
        [flag, report] if flag == "--decode" => match decode(report) {
            Ok(()) => ExitCode::SUCCESS,
            Err(error) => {
                eprintln!("joybusd: {error}");
                ExitCode::FAILURE
            }
        },
        [flag] if flag == "--version" => {
            println!("joybusd {}", env!("CARGO_PKG_VERSION"));
            ExitCode::SUCCESS
        }
        _ => {
            eprintln!("usage: joybusd --decode <hex report>");
            eprintln!("       joybusd --version");
            ExitCode::from(2)
        }
    }
}
