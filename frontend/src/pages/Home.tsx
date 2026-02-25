import { Link } from 'react-router-dom';
import Sticker from '../components/Sticker';

export default function Home() {
  return (
    <div>
      <div className="flex flex-col items-center text-center mb-10">
        <Sticker name="tariffs" className="w-24 h-24 sm:w-32 sm:h-32 mb-6" />
        <h1 className="text-2xl sm:text-3xl font-bold text-white mb-3">
          <span style={{ color: '#08c' }}>Telegram</span> без границ — быстро и безопасно
        </h1>
        <p className="text-gray-400 text-sm sm:text-base leading-relaxed max-w-2xl">
          MTProxy — это официальный протокол Telegram, разработанный командой Дурова специально для обхода блокировок. В отличие от обычных VPN, он не меняет ваш IP полностью и не тормозит интернет — он просто «притворяется» обычным HTTPS-трафиком, оставаясь при этом зашифрованным.
        </p>
      </div>

      <div className="mb-8 max-w-2xl mx-auto">
        <div className="bg-gray-900/60 border border-gray-800 rounded-xl px-5 py-5 sm:px-6 sm:py-6">
          <h2 className="text-base font-semibold text-white mb-4">Наши MTProxy ссылки</h2>
          <ul className="space-y-2.5 text-gray-400 text-sm">
            <li className="flex items-start gap-2.5">
              <span className="text-indigo-400 shrink-0 mt-0.5">✓</span>
              <span>Работают в один клик — вставил ссылку в Telegram, нажал «Подключить»</span>
            </li>
            <li className="flex items-start gap-2.5">
              <span className="text-indigo-400 shrink-0 mt-0.5">✓</span>
              <span>Не требуют установки приложений, настройки роутера или технических знаний</span>
            </li>
            <li className="flex items-start gap-2.5">
              <span className="text-indigo-400 shrink-0 mt-0.5">✓</span>
              <span>Используют шифрование на уровне протокола MTProto 2.0</span>
            </li>
            <li className="flex items-start gap-2.5">
              <span className="text-indigo-400 shrink-0 mt-0.5">✓</span>
              <span>Маскируют трафик под обычный веб — ни один DPI-фильтр не распознает Telegram</span>
            </li>
          </ul>
        </div>
      </div>

      <div className="flex flex-wrap gap-3 justify-center">
        <Link
          to="/pricing"
          className="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg px-5 py-2.5 transition-colors"
        >
          Выбрать тариф
        </Link>
        <Link
          to="/proxies"
          className="bg-gray-800 hover:bg-gray-700 text-gray-200 text-sm font-medium rounded-lg px-5 py-2.5 transition-colors"
        >
          Мои прокси
        </Link>
      </div>
    </div>
  );
}
