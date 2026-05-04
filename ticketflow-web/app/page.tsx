import Link from "next/link";

export default function HomePage() {
  return (
    <main className="min-h-screen bg-surface">
      <section className="mx-auto flex min-h-screen max-w-6xl flex-col justify-center px-4 py-10">
        <div className="max-w-3xl">
          <p className="text-sm font-semibold uppercase tracking-[0.08em] text-primary">
            TicketFlow
          </p>
          <h1 className="mt-4 text-4xl font-semibold leading-tight text-text sm:text-5xl">
            Вебсистема управління запитами клієнтів у сервісі IT-послуг
          </h1>
          <p className="mt-5 max-w-2xl text-base leading-7 text-muted">
            Система централізує створення, опрацювання та контроль заявок,
            підтримує рольовий доступ, ізоляцію даних організацій, SLA та
            роботу з вкладеннями.
          </p>

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <Link
              className="inline-flex justify-center rounded-md bg-primary px-5 py-3 text-sm font-medium text-white transition hover:bg-primary/90"
              href="/login"
            >
              Увійти
            </Link>
            <Link
              className="inline-flex justify-center rounded-md border border-border bg-white px-5 py-3 text-sm font-medium text-text transition hover:bg-white/80"
              href="/register"
            >
              Зареєструвати організацію
            </Link>
          </div>
        </div>

        <div className="mt-12 grid gap-4 md:grid-cols-3">
          <article className="rounded-lg border border-border bg-white p-5">
            <h2 className="text-base font-semibold text-text">Багатоклієнтність</h2>
            <p className="mt-2 text-sm leading-6 text-muted">
              Дані користувачів, заявок і вкладень логічно ізольовані в межах
              організації.
            </p>
          </article>
          <article className="rounded-lg border border-border bg-white p-5">
            <h2 className="text-base font-semibold text-text">Життєвий цикл заявки</h2>
            <p className="mt-2 text-sm leading-6 text-muted">
              Заявки проходять контрольовані статуси від створення до закриття
              або повторного відкриття.
            </p>
          </article>
          <article className="rounded-lg border border-border bg-white p-5">
            <h2 className="text-base font-semibold text-text">SLA-контроль</h2>
            <p className="mt-2 text-sm leading-6 text-muted">
              Час опрацювання розраховується з урахуванням пріоритету та
              активного часу заявки.
            </p>
          </article>
        </div>
      </section>
    </main>
  );
}
