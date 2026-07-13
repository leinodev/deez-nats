// Hand-written (NOT generated): proves Kotlin <-> Go protobuf-binary wire interop.
// GO_ENVELOPE_HEX is the exact byte sequence emitted by Go's
// marshaller.DefaultProtoMarshaller for a fixed AuditEntry (see cmd/xlangdump).
// This program decodes it with the generated Kotlin types, re-encodes, and
// asserts the bytes are identical — i.e. the two languages share one wire format.
@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package ru.lnik801l.example

import kotlinx.serialization.protobuf.ProtoBuf
import ru.lnik801l.example.runtime.DnatsAny
import ru.lnik801l.example.runtime.DnatsWrap

private const val GO_ENVELOPE_HEX =
    "0a470a26747970652e676f6f676c65617069732e636f6d2f6578616d706c652e4175646974456e747279121d0a02613112026f31180122060a016b1201762807316300000000000000"

private const val AUDIT_TYPE_URL = "type.googleapis.com/example.AuditEntry"

fun main() {
    val pb = ProtoBuf { encodeDefaults = false }
    val goBytes = fromHex(GO_ENVELOPE_HEX)

    // 1) Decode the Go-produced envelope with the generated Kotlin types.
    val wrap = pb.decodeFromByteArray(DnatsWrap.serializer(), goBytes)
    check(wrap.error.isEmpty()) { "unexpected envelope error: ${wrap.error}" }
    val any = requireNotNull(wrap.data) { "envelope had no Any payload" }
    check(any.typeUrl == AUDIT_TYPE_URL) { "type_url mismatch: ${any.typeUrl}" }

    val entry = pb.decodeFromByteArray(AuditEntry.serializer(), any.value)
    check(entry.id == "a1") { "id=${entry.id}" }
    check(entry.actorId == "o1") { "actorId=${entry.actorId}" }
    check(entry.action == Action.ACTION_CREATE) { "action=${entry.action}" }
    check(entry.meta["k"] == "v") { "meta=${entry.meta}" }
    check(entry.at == 7L) { "at=${entry.at}" }
    check(entry.seq == 99L) { "seq=${entry.seq}" }
    println("decode(Go bytes) -> $entry  [OK]")

    // 2) Re-encode the same value in Kotlin and compare to the Go bytes.
    val reencoded = pb.encodeToByteArray(
        DnatsWrap.serializer(),
        DnatsWrap(DnatsAny(AUDIT_TYPE_URL, pb.encodeToByteArray(AuditEntry.serializer(), entry)), ""),
    )
    val ktHex = toHex(reencoded)
    println("GO_HEX = $GO_ENVELOPE_HEX")
    println("KT_HEX = $ktHex")
    check(ktHex == GO_ENVELOPE_HEX) { "Kotlin re-encode is NOT byte-identical to Go" }

    println("CROSS-LANGUAGE OK: Kotlin and Go share one protobuf-binary wire (byte-identical).")
}

private fun fromHex(s: String): ByteArray =
    ByteArray(s.length / 2) { ((s[it * 2].digit() shl 4) or s[it * 2 + 1].digit()).toByte() }

private fun Char.digit(): Int = Character.digit(this, 16)

private fun toHex(b: ByteArray): String {
    val sb = StringBuilder(b.size * 2)
    for (x in b) sb.append("0123456789abcdef"[(x.toInt() ushr 4) and 0xF]).append("0123456789abcdef"[x.toInt() and 0xF])
    return sb.toString()
}
