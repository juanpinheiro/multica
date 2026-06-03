// Bare layout for the prototype. Bypasses the workspace shell + auth chrome
// — the prototype is a vacuum of mock data, not the real app.
export default function PrototypeLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="h-full w-full overflow-auto bg-background text-foreground">{children}</div>;
}
