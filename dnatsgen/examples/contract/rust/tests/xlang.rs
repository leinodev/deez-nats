// Hand-written (NOT generated): proves Rust <-> Go/Kotlin protobuf-binary interop.
// GO_ENVELOPE_HEX is the exact byte sequence Go's marshaller.DefaultProtoMarshaller
// emits for a fixed AuditEntry (see cmd/xlangdump; the Go and Kotlin golden tests
// assert the same hex). This decodes it with the generated Rust types, re-encodes
// via the dnats runtime, and asserts the bytes are identical.
use dnats_contract_example::contract::{Action, AuditEntry};
use dnats_contract_example::dnats::{self, ProtobufWrap};
use prost::Message;

const GO_ENVELOPE_HEX: &str = "0a470a26747970652e676f6f676c65617069732e636f6d2f6578616d706c652e4175646974456e747279121d0a02613112026f31180122060a016b1201762807316300000000000000";
const AUDIT_TYPE_URL: &str = "type.googleapis.com/example.AuditEntry";

#[test]
fn cross_lang_golden_envelope() {
    let go_bytes = from_hex(GO_ENVELOPE_HEX);

    // 1) Decode the Go-produced envelope with the generated Rust types.
    let wrap = ProtobufWrap::decode(go_bytes.as_slice()).expect("decode envelope");
    assert!(wrap.error.is_empty(), "unexpected envelope error: {}", wrap.error);
    let any = wrap.data.expect("envelope had no Any payload");
    assert_eq!(any.type_url, AUDIT_TYPE_URL, "type_url mismatch");

    let entry = AuditEntry::decode(any.value.as_slice()).expect("decode AuditEntry");
    assert_eq!(entry.id, "a1");
    assert_eq!(entry.actor_id, "o1");
    assert_eq!(entry.action, Action::Create as i32);
    assert_eq!(entry.meta.get("k").map(String::as_str), Some("v"));
    assert_eq!(entry.at, 7);
    assert_eq!(entry.seq, 99);

    // 2) Re-encode the same value in Rust and compare to the Go bytes.
    let reencoded = dnats::wrap(&entry, AUDIT_TYPE_URL);
    assert_eq!(
        to_hex(&reencoded),
        GO_ENVELOPE_HEX,
        "Rust re-encode is NOT byte-identical to Go"
    );
}

fn from_hex(s: &str) -> Vec<u8> {
    (0..s.len() / 2)
        .map(|i| u8::from_str_radix(&s[i * 2..i * 2 + 2], 16).unwrap())
        .collect()
}

fn to_hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}
