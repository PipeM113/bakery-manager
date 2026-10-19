import { useEffect, useState } from "react";

const QUERY = "(max-width: 767px)"; // below Tailwind's md

/** True on a phone-sized screen; lists are shown as cards there and as tables on bigger screens. */
export function useIsPhone(): boolean {
  const [phone, setPhone] = useState(() => window.matchMedia(QUERY).matches);
  useEffect(() => {
    const mq = window.matchMedia(QUERY);
    const onChange = () => setPhone(mq.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);
  return phone;
}
