/**
 * Stable accent color for the avatar based on the email.
 *
 * Gives each user a personal color even when full_name is generic.
 * Same algo as the topbar ProfileMenu so the two avatars match.
 */
const PALETTE = [
  'linear-gradient(135deg,#38bdf8,#22d3ee)', // cyan
  'linear-gradient(135deg,#a78bfa,#818cf8)', // violet
  'linear-gradient(135deg,#fb923c,#f59e0b)', // amber
  'linear-gradient(135deg,#34d399,#10b981)', // emerald
  'linear-gradient(135deg,#f472b6,#ec4899)', // pink
  'linear-gradient(135deg,#60a5fa,#3b82f6)', // blue
];

export default function avatarColor(seed: string): string {
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) >>> 0;
  }
  return PALETTE[hash % PALETTE.length] || PALETTE[0];
}
