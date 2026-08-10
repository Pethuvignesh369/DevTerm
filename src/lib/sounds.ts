/**
 * Notification sounds using Web Audio API.
 * No external files needed — generates tones programmatically.
 */

let audioCtx: AudioContext | null = null;

function getAudioContext(): AudioContext {
  if (!audioCtx) {
    audioCtx = new AudioContext();
  }
  return audioCtx;
}

function playTone(frequency: number, duration: number, type: OscillatorType = "sine", volume = 0.15) {
  try {
    const ctx = getAudioContext();
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();

    osc.type = type;
    osc.frequency.setValueAtTime(frequency, ctx.currentTime);
    gain.gain.setValueAtTime(volume, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.start(ctx.currentTime);
    osc.stop(ctx.currentTime + duration);
  } catch {
    // Audio not available (e.g., no user interaction yet)
  }
}

/** Happy ascending tone — connection established */
export function playConnectSound() {
  playTone(440, 0.1, "sine", 0.12);
  setTimeout(() => playTone(554, 0.1, "sine", 0.12), 80);
  setTimeout(() => playTone(659, 0.15, "sine", 0.12), 160);
}

/** Descending tone — connection lost */
export function playDisconnectSound() {
  playTone(440, 0.15, "sine", 0.1);
  setTimeout(() => playTone(349, 0.15, "sine", 0.1), 120);
  setTimeout(() => playTone(261, 0.25, "sine", 0.08), 240);
}

/** Quick blip — notification */
export function playNotificationSound() {
  playTone(880, 0.08, "sine", 0.08);
  setTimeout(() => playTone(1100, 0.06, "sine", 0.06), 60);
}

/** Error buzz */
export function playErrorSound() {
  playTone(200, 0.15, "square", 0.06);
  setTimeout(() => playTone(180, 0.2, "square", 0.05), 100);
}
