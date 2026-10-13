import {
  Baby, Banknote, BookOpen, Briefcase, Bus, Car, Cigarette, Coffee, CreditCard, Dumbbell, Film, Fuel,
  Gamepad2, Gift, GraduationCap, HeartPulse, House, Landmark, Music, PawPrint, PiggyBank, Pill, Plane,
  Receipt, Repeat, Scissors, Shirt, ShoppingCart, Smartphone, Sofa, Sparkles, Tag, Utensils, Wifi,
  Wrench, Zap, type LucideIcon,
} from 'lucide-react'

import type { components } from '@/shared/api/generated/schema'

export type CategoryIconName = components['schemas']['CategoryIcon']

// SPEC-0042: the fixed set the API accepts, with a Russian label for each icon.
export const categoryIcons: Record<CategoryIconName, { icon: LucideIcon; label: string }> = {
  'shopping-cart': { icon: ShoppingCart, label: 'Продукты' },
  utensils: { icon: Utensils, label: 'Кафе и рестораны' },
  coffee: { icon: Coffee, label: 'Кофе' },
  car: { icon: Car, label: 'Автомобиль' },
  bus: { icon: Bus, label: 'Транспорт' },
  fuel: { icon: Fuel, label: 'Топливо' },
  home: { icon: House, label: 'Дом' },
  zap: { icon: Zap, label: 'Коммунальные услуги' },
  wifi: { icon: Wifi, label: 'Интернет' },
  smartphone: { icon: Smartphone, label: 'Связь' },
  'heart-pulse': { icon: HeartPulse, label: 'Здоровье' },
  pill: { icon: Pill, label: 'Аптека' },
  shirt: { icon: Shirt, label: 'Одежда' },
  gift: { icon: Gift, label: 'Подарки' },
  'gamepad-2': { icon: Gamepad2, label: 'Игры' },
  film: { icon: Film, label: 'Кино' },
  music: { icon: Music, label: 'Музыка' },
  'book-open': { icon: BookOpen, label: 'Книги' },
  'graduation-cap': { icon: GraduationCap, label: 'Образование' },
  baby: { icon: Baby, label: 'Дети' },
  'paw-print': { icon: PawPrint, label: 'Животные' },
  plane: { icon: Plane, label: 'Путешествия' },
  dumbbell: { icon: Dumbbell, label: 'Спорт' },
  scissors: { icon: Scissors, label: 'Красота' },
  wrench: { icon: Wrench, label: 'Ремонт' },
  'credit-card': { icon: CreditCard, label: 'Карты и комиссии' },
  landmark: { icon: Landmark, label: 'Налоги и госуслуги' },
  receipt: { icon: Receipt, label: 'Счета' },
  repeat: { icon: Repeat, label: 'Подписки' },
  briefcase: { icon: Briefcase, label: 'Работа' },
  banknote: { icon: Banknote, label: 'Доход' },
  'piggy-bank': { icon: PiggyBank, label: 'Накопления' },
  sparkles: { icon: Sparkles, label: 'Развлечения' },
  tag: { icon: Tag, label: 'Прочее' },
  sofa: { icon: Sofa, label: 'Мебель' },
  cigarette: { icon: Cigarette, label: 'Табак' },
}
