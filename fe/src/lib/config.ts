const readApiBase = () => {
  const publicUrl = (import.meta.env.PUBLIC_API_URL || "").trim();
  return publicUrl.replace(/\/+$/, "");
};

export const scrapMapsApiBase = readApiBase();
