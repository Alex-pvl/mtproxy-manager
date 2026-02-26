import { useState, useEffect } from 'react';
import { paymentApi } from '../api/client';
import type { Plan } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';
import Sticker from '../components/Sticker';

const POPULAR_PLAN = 'year_1';

export default function Pricing() {
  const { user, refreshUser } = useAuth();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [buyingPlan, setBuyingPlan] = useState<string | null>(null);
  const [error, setError] = useState('');

  const sub = user?.subscription;

  useEffect(() => {
    paymentApi.listPlans()
      .then((res) => setPlans(res.data))
      .catch(() => setError('Не удалось загрузить тарифы'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.has('payment')) {
      paymentApi
        .checkPendingPayments()
        .then(() => refreshUser())
        .catch(() => refreshUser())
        .finally(() => window.history.replaceState({}, '', '/pricing'));
    }
  }, [refreshUser]);

  const handleBuy = async (planId: string) => {
    setBuyingPlan(planId);
    setError('');
    try {
      const res = await paymentApi.createPayment(planId);
      window.location.href = res.data.payment_url;
    } catch (err: any) {
      setError(err.response?.data?.error || 'Ошибка при создании платежа');
      setBuyingPlan(null);
    }
  };

  if (loading) {
    return <div className="text-gray-400">Загрузка...</div>;
  }

  return (
    <div>
      <div className="text-center mb-6 sm:mb-8">
        <h1 className="text-xl sm:text-2xl font-bold text-white mb-2">Тарифы</h1>
        <p className="text-gray-400 text-sm sm:text-base px-2">Выберите подходящий план для использования MTProxy</p>
      </div>

      <div className="mb-8">
        <h2 className="text-lg font-bold text-white mb-3">Причины пользоваться MTProxy</h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex gap-3">
            <Sticker name="no_logs" className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="text-indigo-400 font-semibold text-sm mb-0.5">Без логов</h3>
              <p className="text-gray-400 text-xs">Защищает вас от слежки провайдеров.</p>
            </div>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex gap-3">
            <Sticker name="cipher" className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="text-indigo-400 font-semibold text-sm mb-0.5">Шифрование</h3>
              <p className="text-gray-400 text-xs">Данные защищены от перехвата и взлома.</p>
            </div>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex gap-3">
            <Sticker name="speed" className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="text-indigo-400 font-semibold text-sm mb-0.5">Быстрая сеть</h3>
              <p className="text-gray-400 text-xs">Серверы имеют скорость соединения до 1 Гбит/с.</p>
            </div>
          </div>
        </div>
      </div>

      {sub?.active && (
        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-lg px-4 py-3 mb-6 text-center">
          <p className="text-emerald-400 text-sm">
            Активная подписка: <span className="font-semibold">{sub.plan_name}</span>
            {sub.expires_at && (
              <span className="text-emerald-500 ml-2">
                до {new Date(sub.expires_at).toLocaleDateString('ru-RU')}
              </span>
            )}
          </p>
        </div>
      )}

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 text-sm rounded px-3 py-2 mb-6 text-center">
          {error}
          <button onClick={() => setError('')} className="ml-2 text-red-300 hover:text-white">&times;</button>
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        {plans.map((plan) => {
          const isPopular = plan.id === POPULAR_PLAN;
          return (
            <div
              key={plan.id}
              className={`relative bg-gray-900 border rounded-lg p-5 flex flex-col ${
                isPopular
                  ? 'border-indigo-500 ring-1 ring-indigo-500/50'
                  : 'border-gray-800'
              }`}
            >
              {isPopular && (
                <span className="absolute -top-2.5 left-1/2 -translate-x-1/2 bg-indigo-600 text-white text-xs font-medium px-3 py-0.5 rounded-full">
                  Популярный
                </span>
              )}

              <h3 className="text-lg font-semibold text-white mb-1">{plan.name}</h3>

              <div className="mb-4 flex items-baseline gap-2 flex-wrap">
                <span className="text-2xl font-bold text-white">{plan.price_label}</span>
                {plan.original_price_label && (
                  <>
                    <span className="text-base text-gray-500 line-through">{plan.original_price_label}</span>
                    {plan.discount_percent != null && plan.discount_percent > 0 && (
                      <span className="text-sm font-medium text-blue-400">−{plan.discount_percent}%</span>
                    )}
                  </>
                )}
              </div>

              <p className="text-sm text-gray-400 mb-4">
                {plan.per_month}/мес
              </p>

              <ul className="text-sm text-gray-300 space-y-2 mb-6 flex-1">
                <li className="flex items-center gap-2">
                  <span className="text-emerald-400">&#10003;</span>
                  {plan.max_proxies === 1 ? '1 прокси' : `До ${plan.max_proxies} прокси`}
                </li>
              </ul>

              <button
                onClick={() => handleBuy(plan.id)}
                disabled={buyingPlan !== null}
                className={`w-full text-sm font-medium rounded px-4 py-2.5 transition-colors disabled:opacity-50 touch-manipulation ${
                  isPopular
                    ? 'bg-indigo-600 hover:bg-indigo-500 text-white'
                    : 'bg-gray-800 hover:bg-gray-700 text-gray-200'
                }`}
              >
                {buyingPlan === plan.id
                  ? 'Перенаправление...'
                  : sub?.active
                    ? 'Продлить'
                    : 'Купить'}
              </button>
            </div>
          );
        })}
      </div>

      <div className="mt-8 text-center">
        <Link to="/proxies" className="text-sm text-gray-400 hover:text-white transition-colors">
          &larr; Вернуться к панели
        </Link>
      </div>
    </div>
  );
}
