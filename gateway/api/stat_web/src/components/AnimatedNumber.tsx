import { useEffect, useRef, useState } from "react";
import { formatNumber } from "../lib/format";

interface AnimatedNumberProps {
  value: number;
  reducedMotion?: boolean;
  duration?: number;
}

export function AnimatedNumber({ value, reducedMotion = false, duration = 720 }: AnimatedNumberProps) {
  const target = Number.isFinite(value) ? value : 0;
  const [displayValue, setDisplayValue] = useState(() => reducedMotion ? target : 0);
  const currentValue = useRef(0);
  const frame = useRef<number | null>(null);

  useEffect(() => {
    if (frame.current !== null) {
      window.cancelAnimationFrame(frame.current);
      frame.current = null;
    }

    if (reducedMotion || currentValue.current === target) {
      currentValue.current = target;
      setDisplayValue(target);
      return;
    }

    const startValue = currentValue.current;
    const startTime = window.performance.now();
    const animationDuration = Math.max(1, duration);
    const tick = (time: number) => {
      const progress = Math.min(1, Math.max(0, (time - startTime) / animationDuration));
      const easedProgress = 1 - (1 - progress) ** 3;
      const nextValue = startValue + (target - startValue) * easedProgress;
      currentValue.current = nextValue;
      setDisplayValue(nextValue);

      if (progress < 1) {
        frame.current = window.requestAnimationFrame(tick);
      } else {
        currentValue.current = target;
        setDisplayValue(target);
        frame.current = null;
      }
    };

    frame.current = window.requestAnimationFrame(tick);
    return () => {
      if (frame.current !== null) window.cancelAnimationFrame(frame.current);
      frame.current = null;
    };
  }, [duration, reducedMotion, target]);

  return <span>{formatNumber(Math.round(displayValue))}</span>;
}
