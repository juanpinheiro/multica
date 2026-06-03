export default function ChromePrototypeLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="h-full w-full bg-background text-foreground">{children}</div>;
}
