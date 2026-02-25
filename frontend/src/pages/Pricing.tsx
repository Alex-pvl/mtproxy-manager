import { useState, useEffect } from 'react';
import { paymentApi } from '../api/client';
import type { Plan } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';

const POPULAR_PLAN = 'month_3';

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
      refreshUser();
      window.history.replaceState({}, '', '/pricing');
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
      <div className="text-center mb-8">
        <h1 className="text-2xl font-bold text-white mb-2">Тарифы</h1>
        <p className="text-gray-400">Выберите подходящий план для использования MTProxy</p>
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

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
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

              <div className="mb-4">
                <span className="text-2xl font-bold text-white">{plan.price_label}</span>
              </div>

              <p className="text-sm text-gray-400 mb-4">
                {plan.per_month}/мес
              </p>

              <ul className="text-sm text-gray-300 space-y-2 mb-6 flex-1">
                <li className="flex items-center gap-2">
                  <span className="text-emerald-400">&#10003;</span>
                  До {plan.max_proxies} прокси
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-emerald-400">&#10003;</span>
                  MTProto протокол
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-emerald-400">&#10003;</span>
                  Маскировка трафика
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-emerald-400">&#10003;</span>
                  Панель управления
                </li>
              </ul>

              <button
                onClick={() => handleBuy(plan.id)}
                disabled={buyingPlan !== null}
                className={`w-full text-sm font-medium rounded px-4 py-2.5 transition-colors disabled:opacity-50 ${
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
        <Link to="/" className="text-sm text-gray-400 hover:text-white transition-colors">
          &larr; Вернуться к панели
        </Link>
      </div>
    </div>
  );
}
