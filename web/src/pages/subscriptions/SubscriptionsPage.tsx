import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { ReminderHistory } from '@/features/payment-reminders/ReminderHistory'
import { csrfToken } from '@/shared/model/auth-session'
import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarClock, Pause, Pencil, Plus, Repeat2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/shared/api/client'
import type { components } from '@/shared/api/generated/schema'
import { dateText, eventDateText, formatMoney, parseMinor } from '@/shared/lib/finance-display'
import { AppLayout } from '@/shared/ui/layout/AppLayout'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Card, CardContent } from '@/shared/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/shared/ui/dialog'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { CreateFinancialEventForm } from '@/features/create-financial-event'

type Subscription = components['schemas']['Subscription']
type InputData = components['schemas']['SubscriptionInput']
const periodLabels = { weekly: 'Каждую неделю', monthly: 'Каждый месяц', yearly: 'Каждый год' }
export function SubscriptionsPage() {
 const query = useQueryClient()
 const [edit, setEdit] = useState<Subscription | 'new' | null>(null)
 const [expense, setExpense] = useState<Subscription | null>(null)
 const [purchaseOpen, setPurchaseOpen] = useState(false)
 const [eventID, setEventID] = useState('')
 const subscriptions = useQuery({ queryKey: ['subscriptions'], queryFn: async () => {
  const { data, error } = await apiClient.GET('/api/v1/subscriptions'); if(error) throw new Error(error.message); return data?.items ?? []
 } })
 const accounts = useQuery({ queryKey: ['accounts'], queryFn: async () => { const {data,error}=await apiClient.GET('/api/v1/accounts');if(error)throw new Error(error.message);return data?.items??[] } })
 const events = useQuery({ queryKey: ['financial-events','subscription-picker'], queryFn: async () => { const {data,error}=await apiClient.GET('/api/v1/financial-events');if(error)throw new Error(error.message);return data?.items??[] } })
 const expenseHistory = useQuery({queryKey:['subscription-expenses',expense?.id],enabled:expense!==null,queryFn:async()=>{const {data,error}=await apiClient.GET('/api/v1/subscriptions/{subscriptionId}/expenses',{params:{path:{subscriptionId:expense!.id}}});if(error)throw new Error(error.message);return data?.items??[]}})
 const categories = useQuery({ queryKey: ['categories'], queryFn: async () => { const {data,error}=await apiClient.GET('/api/v1/categories');if(error)throw new Error(error.message);return data?.items??[] } })
 const refresh = () => query.invalidateQueries({queryKey:['subscriptions']})
 const pause = useMutation({mutationFn: async (s:Subscription) => {
  const {error}=await apiClient.PATCH('/api/v1/subscriptions/{subscriptionId}',{params:{header:{'X-CSRF-Token':csrfToken},path:{subscriptionId:s.id}},body:{name:s.name,amountMinor:s.amountMinor,currency:s.currency,period:s.period,startsOn:s.startsOn,endsOn:s.endsOn,accountId:s.accountId,paymentInstrumentId:s.paymentInstrumentId,reminders:s.reminders,paused:!s.paused,revision:s.revision}});if(error)throw new Error(error.message)
 },onSuccess:refresh,onError:e=>toast.error(e.message)})
 const link = useMutation({mutationFn:async()=>{if(!expense) return;const {error}=await apiClient.POST('/api/v1/subscriptions/{subscriptionId}/expenses',{params:{header:{'X-CSRF-Token':csrfToken},path:{subscriptionId:expense.id}},body:{financialEventId:eventID}});if(error)throw new Error(error.message)},onSuccess:()=>{setExpense(null);setEventID('');void refresh();toast.success('Расход связан с подпиской')},onError:e=>toast.error(e.message)})
 const used = new Set(subscriptions.data?.flatMap(s=>s.expenseIds)??[])
 const candidates = events.data?.filter(e=>e.kind==='purchase'&&!used.has(e.id)&&e.postings.some(p=>p.currency===expense?.currency))??[]
 return <AppLayout><div className="space-y-6">
  <div className="flex flex-wrap items-start justify-between gap-4"><div><h1>Подписки</h1><p className="mt-2 text-muted-foreground">Регулярные списания, карты и фактические расходы.</p></div><Button onClick={()=>setEdit('new')}><Plus className="size-4"/>Добавить подписку</Button></div>
  <p className="text-sm text-muted-foreground">Планируемые списания не изменяют остатки. Подтверждённые покупки учитываются в статистике один раз. Напоминания приходят за 3 и 1 день на подтверждённую почту.</p>
  {subscriptions.isPending?<LoadingSkeleton label="Загружаем подписки…"/>:subscriptions.isError?<Card><CardContent className="space-y-4 pt-5"><p role="alert">{subscriptions.error.message}</p><Button variant="outline" onClick={()=>void subscriptions.refetch()}>Повторить</Button></CardContent></Card>:!subscriptions.data?.length?<Card><CardContent className="py-12 text-center"><Repeat2 className="mx-auto mb-4 size-10 text-primary"/><h2>Все регулярные расходы в одном месте</h2><p className="mt-2 text-muted-foreground">Добавьте связь, сервис или другую подписку.</p></CardContent></Card>:<div className="grid gap-4 md:grid-cols-2">{subscriptions.data.map(s=><Card key={s.id}><CardContent className="space-y-4 pt-5"><div className="flex justify-between gap-3"><div><h2 className="text-lg">{s.name}</h2><p className="mt-1 text-2xl font-semibold tabular-nums">{formatMoney(s.amountMinor,s.currency)}</p><p className="mt-1 text-sm text-muted-foreground">{periodLabels[s.period]}</p></div><Button variant="ghost" size="icon" aria-label={`Редактировать ${s.name}`} onClick={()=>setEdit(s)}><Pencil className="size-4"/></Button></div><p className="flex items-center gap-2 text-sm"><CalendarClock className="size-4 text-primary"/>{s.paused?'Приостановлена':s.nextDueOn?`Следующее списание: ${dateText(s.nextDueOn)}`:'Период завершён'}</p><p className="text-sm text-muted-foreground">{subscriptionCard(s,accounts.data??[])} · {s.reminders?'Напоминания включены':'Без напоминаний'}</p><div className="flex flex-wrap gap-2"><Button variant="outline" onClick={()=>{setExpense(s);setEventID('')}}>Расходы · {s.expenseIds.length}</Button><Button variant="ghost" disabled={pause.isPending} onClick={()=>pause.mutate(s)}>{s.paused?<Repeat2 className="size-4"/>:<Pause className="size-4"/>}{s.paused?'Возобновить':'Приостановить'}</Button></div></CardContent></Card>)}</div>}
  <ReminderHistory/>
  <Dialog open={edit!==null} onOpenChange={open=>{if(!open)setEdit(null)}}><DialogContent><DialogHeader><DialogTitle>{edit==='new'?'Новая подписка':'Редактировать подписку'}</DialogTitle><DialogDescription>Ожидаемое списание и необязательная карта. Суммы хранятся в валюте подписки.</DialogDescription></DialogHeader>{edit&&<SubscriptionForm key={edit==='new'?'new':edit.id} initial={edit==='new'?undefined:edit} accounts={accounts.data??[]} onSaved={()=>{setEdit(null);void refresh()}}/>}</DialogContent></Dialog>
  <Dialog open={expense!==null} onOpenChange={open=>{if(!open)setExpense(null)}}><DialogContent><DialogHeader><DialogTitle>Расходы: {expense?.name}</DialogTitle><DialogDescription>Свяжите уже созданную покупку. Это не создаёт повторное списание.</DialogDescription></DialogHeader><ExpenseHistory loading={expenseHistory.isPending} error={expenseHistory.error} events={expenseHistory.data??[]}/><p className="text-sm text-muted-foreground">Связано покупок: {expense?.expenseIds.length??0}. Ниже — до 100 последних операций.</p>{events.isError?<p role="alert">{events.error.message}</p>:<ChoiceSelect aria-label="Фактическая покупка" value={eventID} onValueChange={setEventID}><option value="">Выберите покупку</option>{candidates.map(e=><option value={e.id} key={e.id}>{e.merchant||e.description||'Покупка'} · {formatMoney(e.postings[0].amountMinor,e.postings[0].currency)}</option>)}</ChoiceSelect>}<Button disabled={!eventID||link.isPending} onClick={()=>link.mutate()}>Связать расход</Button><Button variant="outline" onClick={()=>setPurchaseOpen(true)}>Создать новую покупку</Button></DialogContent></Dialog>
  <Dialog open={purchaseOpen} onOpenChange={setPurchaseOpen}><DialogContent><DialogHeader><DialogTitle>Новая покупка</DialogTitle><DialogDescription>После подтверждения выберите покупку в списке и свяжите её с подпиской.</DialogDescription></DialogHeader><CreateFinancialEventForm accounts={accounts.data??[]} categories={categories.data??[]} events={events.data??[]} onSaved={()=>{setPurchaseOpen(false);void events.refetch()}}/></DialogContent></Dialog>
 </div></AppLayout>
}
function SubscriptionForm({initial,accounts,onSaved}:{initial?:Subscription;accounts:components['schemas']['Account'][];onSaved:()=>void}){
 const creationKey=useRef(createIdempotencyKey())
 const [period,setPeriod]=useState<InputData['period']>(initial?.period??'monthly')
 const [account,setAccount]=useState(initial?.paymentInstrumentId??initial?.accountId??'')
 const [reminders,setReminders]=useState(initial?.reminders??false)
 const save=useMutation({mutationFn:async(form:FormData)=>{
  const amount=parseMinor(String(form.get('amount')));if(amount===null||amount<=0)throw new Error('Укажите положительную сумму с точностью до копейки.')
  const body:InputData={name:String(form.get('name')),amountMinor:amount,currency:String(form.get('currency')).toUpperCase(),period,startsOn:String(form.get('start')),endsOn:String(form.get('end'))||null,accountId:accounts.find(a=>a.id===account||a.paymentInstruments?.some(c=>c.id===account))?.id??null,paymentInstrumentId:accounts.flatMap(a=>a.paymentInstruments??[]).find(c=>c.id===account)?.id??null,paused:initial?.paused??false,reminders,revision:initial?.revision??0}
  const result=initial?await apiClient.PATCH('/api/v1/subscriptions/{subscriptionId}',{params:{header:{'X-CSRF-Token':csrfToken},path:{subscriptionId:initial.id}},body}):await apiClient.POST('/api/v1/subscriptions',{params:{header:{'X-CSRF-Token':csrfToken,'Idempotency-Key':creationKey.current}},body});if(result.error)throw new Error(result.error.message)
 },onSuccess:()=>{toast.success('Подписка сохранена');onSaved()},onError:e=>toast.error(e.message)})
 return <form className="space-y-4" onSubmit={e=>{e.preventDefault();save.mutate(new FormData(e.currentTarget))}}>
  <div className="space-y-2"><Label htmlFor="sub-name">Название</Label><Input id="sub-name" name="name" defaultValue={initial?.name} required maxLength={160}/></div>
  <div className="grid gap-4 sm:grid-cols-2"><div className="space-y-2"><Label htmlFor="sub-amount">Сумма</Label><Input id="sub-amount" name="amount" inputMode="decimal" defaultValue={initial?String(initial.amountMinor/100):''} required/></div><div className="space-y-2"><Label htmlFor="sub-currency">Валюта</Label><Input id="sub-currency" name="currency" defaultValue={initial?.currency??'RUB'} required maxLength={3} pattern="[A-Za-z]{3}"/></div></div>
  <div className="space-y-2"><Label htmlFor="sub-period">Период</Label><ChoiceSelect id="sub-period" value={period} onValueChange={v=>setPeriod(v as InputData['period'])}>{Object.entries(periodLabels).map(([v,label])=><option key={v} value={v}>{label}</option>)}</ChoiceSelect></div>
  <div className="grid gap-4 sm:grid-cols-2"><div className="space-y-2"><Label htmlFor="sub-start">Первое списание</Label><Input id="sub-start" name="start" type="date" defaultValue={initial?.startsOn} required/></div><div className="space-y-2"><Label htmlFor="sub-end">До даты · необязательно</Label><Input id="sub-end" name="end" type="date" defaultValue={initial?.endsOn??''}/></div></div>
  <div className="space-y-2"><Label htmlFor="sub-account">С какой карты или счёта</Label><ChoiceSelect id="sub-account" value={account} onValueChange={setAccount}><option value="">Не указывать</option>{accounts.flatMap(a=>[<option key={a.id} value={a.id}>{a.name} · счёт</option>,...(a.paymentInstruments??[]).map(c=><option key={c.id} value={c.id}>{c.bank} *{c.last4}</option>)])}</ChoiceSelect></div>
  <label className="flex min-h-11 items-start gap-3 text-sm"><Checkbox checked={reminders} onCheckedChange={v=>setReminders(v===true)}/><span>Напоминать по почте за 3 и 1 день. Нужна подтверждённая почта в профиле.</span></label>
  <Button className="w-full" disabled={save.isPending}>{save.isPending?'Сохраняю…':'Сохранить'}</Button>
 </form>
}

function subscriptionCard(s:Subscription,accounts:components['schemas']['Account'][]){
 const account=accounts.find(a=>a.id===s.accountId)
 if(s.paymentInstrumentId){const card=account?.paymentInstruments?.find(c=>c.id===s.paymentInstrumentId);return card?`${card.bank} *${card.last4}`:'Карта недоступна'}
 return account?account.name:s.accountId?'Счёт недоступен':'Карта не указана'
}
function ExpenseHistory({loading,error,events}:{loading:boolean;error:Error|null;events:components['schemas']['FinancialEvent'][]}){
 if(loading)return <LoadingSkeleton label="Загружаем расходы…"/>
 if(error)return <p role="alert">{error.message}</p>
 if(!events.length)return <p className="text-sm text-muted-foreground">Подтверждённых расходов пока нет.</p>
 return <section className="space-y-3"><h3 className="font-medium">Последние связанные покупки · до 100</h3>{events.map(e=><div key={e.id} className="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-muted/50 p-3 text-sm"><div><p>{e.merchant||e.description||'Покупка'}</p><p className="text-xs text-muted-foreground">{eventDateText(e)}</p></div><div className="tabular-nums">{e.postings.filter(p=>p.direction==='debit').map(p=><p key={p.id}>{formatMoney(p.amountMinor,p.currency)}</p>)}</div></div>)}</section>
}
