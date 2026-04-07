/**
 * Root dialog memakai atribut HTML `hidden` (lihat Dialog.astro), bukan hanya kelas Tailwind.
 * Wajib pakai removeAttribute/setAttribute — classList `hidden` tidak memengaruhi atribut tersebut.
 */
export function openModal(id: string): void {
  const el = document.getElementById(id);
  if (!el) return;
  el.removeAttribute("hidden");
  el.classList.add("flex", "items-center", "justify-center");
  el.setAttribute("aria-hidden", "false");
}

export function closeModal(id: string): void {
  const el = document.getElementById(id);
  if (!el) return;
  el.setAttribute("hidden", "");
  el.classList.remove("flex", "items-center", "justify-center");
  el.setAttribute("aria-hidden", "true");
}

/** Panel geser (Sheets.astro): backdrop + drawer, tanpa center seperti modal. */
export function openSheet(id: string): void {
  const el = document.getElementById(id);
  if (!el) return;
  el.removeAttribute("hidden");
  el.classList.add("flex", "items-stretch", "justify-start");
  el.setAttribute("aria-hidden", "false");
  document.body.style.overflow = "hidden";
}

export function closeSheet(id: string): void {
  const el = document.getElementById(id);
  if (!el) return;
  el.setAttribute("hidden", "");
  el.classList.remove("flex", "items-stretch", "justify-start");
  el.setAttribute("aria-hidden", "true");
  document.body.style.overflow = "";
}
