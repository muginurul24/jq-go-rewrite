import { motion, useReducedMotion } from "framer-motion"

const streams = [
  { left: "4%", delay: 0.1, duration: 13, content: "010100110101" },
  { left: "11%", delay: 0.8, duration: 15, content: "111001010011" },
  { left: "19%", delay: 1.5, duration: 12, content: "001101001110" },
  { left: "27%", delay: 0.3, duration: 14, content: "101011001001" },
  { left: "35%", delay: 1.8, duration: 16, content: "011100101010" },
  { left: "44%", delay: 0.5, duration: 11, content: "110010101101" },
  { left: "53%", delay: 2.1, duration: 14, content: "000111010011" },
  { left: "62%", delay: 0.9, duration: 17, content: "101001110100" },
  { left: "71%", delay: 1.2, duration: 13, content: "011011001110" },
  { left: "80%", delay: 0.4, duration: 15, content: "100101011001" },
  { left: "89%", delay: 1.6, duration: 12, content: "010011101101" },
]

export function MatrixBackdrop() {
  const reducedMotion = useReducedMotion()

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 overflow-hidden rounded-[inherit]"
    >
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.2),transparent_38%),radial-gradient(circle_at_top_right,rgba(14,165,233,0.18),transparent_36%),linear-gradient(180deg,rgba(3,7,18,0)_0%,rgba(3,7,18,0.24)_100%)] dark:bg-[radial-gradient(circle_at_top_left,rgba(52,211,153,0.22),transparent_42%),radial-gradient(circle_at_top_right,rgba(56,189,248,0.18),transparent_36%),linear-gradient(180deg,rgba(2,6,23,0.18)_0%,rgba(2,6,23,0.58)_100%)]" />
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(15,23,42,0.05)_1px,transparent_1px),linear-gradient(to_bottom,rgba(15,23,42,0.05)_1px,transparent_1px)] bg-[size:3.5rem_3.5rem] dark:bg-[linear-gradient(to_right,rgba(148,163,184,0.08)_1px,transparent_1px),linear-gradient(to_bottom,rgba(148,163,184,0.08)_1px,transparent_1px)]" />
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-emerald-400/70 to-transparent" />
      {!reducedMotion &&
        streams.map((stream) => (
          <motion.div
            key={stream.left}
            className="absolute top-0 flex h-52 w-8 -translate-y-full items-start justify-center overflow-hidden text-[10px] leading-[1.15] font-mono text-emerald-500/25 dark:text-emerald-300/30"
            style={{ left: stream.left }}
            animate={{ y: ["-20%", "160%"], opacity: [0, 0.9, 0.3, 0] }}
            transition={{
              duration: stream.duration,
              delay: stream.delay,
              repeat: Number.POSITIVE_INFINITY,
              ease: "linear",
            }}
          >
            <span
              className="tracking-[0.45em]"
              style={{ writingMode: "vertical-rl" }}
            >
              {stream.content}
            </span>
          </motion.div>
        ))}
      {!reducedMotion && (
        <motion.div
          className="absolute inset-x-8 top-24 h-px bg-gradient-to-r from-transparent via-emerald-300/70 to-transparent blur-sm"
          animate={{ opacity: [0.15, 0.55, 0.15], y: [0, 10, 0] }}
          transition={{
            duration: 4,
            repeat: Number.POSITIVE_INFINITY,
            ease: "easeInOut",
          }}
        />
      )}
    </div>
  )
}
