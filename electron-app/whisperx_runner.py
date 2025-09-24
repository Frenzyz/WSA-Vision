#!/usr/bin/env python3
import argparse
import json
import sys
import os
import tempfile

def main():
    parser = argparse.ArgumentParser(description="WhisperX transcription runner")
    parser.add_argument('audio_path', help='Path to audio file (wav/mp3/webm/etc)')
    parser.add_argument('--model', default='small', help='WhisperX ASR model (e.g., large-v2)')
    parser.add_argument('--device', default='cpu', help='Device: cuda or cpu')
    parser.add_argument('--compute_type', default='int8', help='Compute type for faster-whisper backend (float16/int8)')
    parser.add_argument('--batch_size', type=int, default=8, help='Batch size')
    parser.add_argument('--no_align', action='store_true', default=True, help='Disable alignment step to avoid downloads/TLS issues')
    parser.add_argument('--language', default='', help='Optional language code, e.g., en, de, fr')
    args = parser.parse_args()

    try:
        import whisperx
    except Exception as e:
        sys.stderr.write(
            "WhisperX is not installed. Install with: pip install -U whisperx\n"
        )
        sys.stderr.write(str(e) + "\n")
        # Return JSON error instead of non-zero exit
        sys.stdout.write(json.dumps({"text": "", "error": f"import whisperx failed: {e}"}))
        sys.exit(0)

    # Optional: convert to wav for better compatibility (webm/opus, etc.)
    src_audio = args.audio_path
    tmp_wav = None
    ext = os.path.splitext(src_audio)[1].lower()
    if ext in {'.webm', '.ogg', '.m4a', '.mp4'}:
        try:
            import av  # PyAV
            with av.open(src_audio) as container:
                stream = next((s for s in container.streams if s.type == 'audio'), None)
                if stream is None:
                    raise RuntimeError('No audio stream found')
                tmp_fd, tmp_path = tempfile.mkstemp(suffix='.wav')
                os.close(tmp_fd)
                out = av.open(tmp_path, mode='w')
                out_stream = out.add_stream('pcm_s16le', rate=16000, layout='mono')
                resampler = av.audio.resampler.AudioResampler(format='s16', layout='mono', rate=16000)
                for frame in container.decode(stream):
                    frame = resampler.resample(frame)
                    packet = out_stream.encode(frame)
                    if packet:
                        out.mux(packet)
                # Flush encoder
                packet = out_stream.encode(None)
                if packet:
                    out.mux(packet)
                out.close()
                src_audio = tmp_path
                tmp_wav = tmp_path
        except Exception as conv_err:
            # If conversion fails, proceed with original file but return warning in text
            src_audio = args.audio_path

    try:
        device = args.device
        compute_type = args.compute_type
        model_name = args.model

        # 1) Transcribe
        model = whisperx.load_model(model_name, device, compute_type=compute_type)
        audio = whisperx.load_audio(src_audio)
        transcribe_kwargs = {"batch_size": args.batch_size}
        if args.language:
            transcribe_kwargs["language"] = args.language
        result = model.transcribe(audio, **transcribe_kwargs)

        # 2) Align (optional)
        lang = result.get("language") or args.language or None
        segments = result.get("segments", [])
        if not args.no_align:
            try:
                model_a, metadata = whisperx.load_align_model(language_code=lang, device=device)
                aligned = whisperx.align(segments, model_a, metadata, audio, device, return_char_alignments=False)
                segments = aligned.get("segments", [])
            except Exception:
                # Alignment optional; fall back to original segments
                pass

        # Concatenate text
        text = " ".join([s.get("text", "").strip() for s in segments]).strip()

        out = {
            "text": text,
            "language": lang,
            "segments": segments,
        }
        sys.stdout.write(json.dumps(out, ensure_ascii=False))
        sys.exit(0)

    except Exception as e:
        # Return JSON error but keep exit code 0 so Electron can handle gracefully
        out = {"text": "", "error": str(e)}
        sys.stdout.write(json.dumps(out, ensure_ascii=False))
        sys.exit(0)
    finally:
        if tmp_wav and os.path.exists(tmp_wav):
            try:
                os.remove(tmp_wav)
            except Exception:
                pass

if __name__ == '__main__':
    main()


