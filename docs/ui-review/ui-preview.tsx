import { createRoot } from 'react-dom/client'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ThemeProvider } from 'next-themes'
import { apiClient } from '@/shared/api/client'
import { AuthContext } from '@/shared/model/auth-context'
import type { StatisticsCurrency } from '@/shared/api/types'
import { HomePage } from '@/pages/home/ui/HomePage'
import { ImportsPage } from '@/pages/imports/ImportsPage'
import { AssistantPage } from '@/pages/assistant/ui/AssistantPage'
import { LoginPage } from '@/pages/login/LoginPage'
import { PasswordEntry } from '@/pages/login/PasswordEntry'
import '@/app/styles/index.css'
const item: StatisticsCurrency = {
 currency:'RUB',summary:{currency:'RUB',openingBalanceMinor:null,closingBalanceMinor:null,incomeMinor:48000000,purchasesMinor:29350000,refundsMinor:150000,debtPaymentsMinor:6000000,netExternalCashFlowMinor:12800000,internalTransferVolumeMinor:5000000,expenseCategories:[]},
 categories:[{categoryId:'food',categoryName:'Продукты',purchasesMinor:12500000,refundsMinor:150000,netExpenseMinor:12350000,shareBasisPoints:4258},{categoryId:'home',categoryName:'Дом и связь',purchasesMinor:8200000,refundsMinor:0,netExpenseMinor:8200000,shareBasisPoints:2793},{categoryId:'travel',categoryName:'Транспорт',purchasesMinor:4300000,refundsMinor:0,netExpenseMinor:4300000,shareBasisPoints:1465},{categoryId:null,categoryName:'Без категории',purchasesMinor:4350000,refundsMinor:0,netExpenseMinor:4350000,shareBasisPoints:1482}],
 series:[
 {from:'2026-05-01',to:'2026-06-01',incomeMinor:8000000,purchasesMinor:5100000,refundsMinor:0,debtPaymentsMinor:1000000,outflowsMinor:6100000},
 {from:'2026-06-01',to:'2026-07-01',incomeMinor:8000000,purchasesMinor:4800000,refundsMinor:0,debtPaymentsMinor:1000000,outflowsMinor:5800000},
 {from:'2026-07-01',to:'2026-08-01',incomeMinor:8000000,purchasesMinor:6200000,refundsMinor:150000,debtPaymentsMinor:1000000,outflowsMinor:7200000},
 {from:'2026-08-01',to:'2026-09-01',incomeMinor:8000000,purchasesMinor:5750000,refundsMinor:0,debtPaymentsMinor:1000000,outflowsMinor:6750000},
 {from:'2026-09-01',to:'2026-10-01',incomeMinor:8000000,purchasesMinor:4900000,refundsMinor:0,debtPaymentsMinor:1000000,outflowsMinor:5900000},
 {from:'2026-10-01',to:'2026-10-10',incomeMinor:8000000,purchasesMinor:2600000,refundsMinor:0,debtPaymentsMinor:1000000,outflowsMinor:3600000},
 ]}
const account={id:'10000000-0000-4000-8000-000000000001',name:'Основной счёт',type:'debit',currency:'RUB',balanceMinor:5800000,balanceReconciled:true,createdAt:'2026-09-01T00:00:00Z',paymentInstruments:[{id:'card',name:'Основная карта',bank:'Банк',last4:'1024'}]}
const accounts=[account,{...account,id:'10000000-0000-4000-8000-000000000002',name:'Кредитная карта',type:'credit',balanceMinor:null,balanceReconciled:false,paymentInstruments:[]}]
const categories=[{id:'food',name:'Продукты',kind:'expense'},{id:'coffee',name:'Кафе',kind:'expense'},{id:'income',name:'Доход',kind:'income'}]
const events=[{id:'purchase',kind:'purchase',source:'manual',categoryId:'food',categoryName:'Продукты',merchant:'Продуктовый магазин',description:'Покупка продуктов',occurredAt:'2026-10-09T10:00:00+05:00',createdAt:'2026-10-09T10:00:00+05:00',postings:[{id:'posting',accountId:account.id,accountName:'Основная карта',direction:'debit',amountMinor:128000,currency:'RUB'}]},{id:'coffee',kind:'purchase',source:'manual',categoryId:'coffee',categoryName:'Кафе',merchant:'Кофейня',description:'Кофе',occurredAt:'2026-10-08T10:00:00+05:00',createdAt:'2026-10-08T10:00:00+05:00',postings:[{id:'posting2',accountId:account.id,accountName:'Основная карта',direction:'debit',amountMinor:24000,currency:'RUB'}]}]
const payments=[{id:'internet',position:1,dueOn:'2026-10-12',amountMinor:82000,paidMinor:0,remainingMinor:82000,status:'pending',isOverdue:false,settlements:[]},{id:'rent',position:2,dueOn:'2026-10-15',amountMinor:1450000,paidMinor:0,remainingMinor:1450000,status:'pending',isOverdue:false,settlements:[]}]
const obligations=[{id:'internet',name:'Интернет',kind:'other',currency:'RUB',totalMinor:82000,paidMinor:0,remainingMinor:82000,status:'active',createdAt:'2026-10-01T00:00:00Z',payments:[payments[0]]},{id:'rent',name:'Аренда',kind:'other',currency:'RUB',totalMinor:1450000,paidMinor:0,remainingMinor:1450000,status:'active',createdAt:'2026-10-01T00:00:00Z',payments:[payments[1]]}]
const cash={asOf:'2026-10-09',through:'2026-10-31',currencies:[{currency:'RUB',liquidBalanceMinor:5800000,overdueObligationsMinor:0,upcomingObligationsMinor:1532000,obligationsThroughMinor:1532000,availableAfterObligationsMinor:4268000,scheduledPayments:payments.map((p,i)=>({...p,scheduledPaymentId:p.id,obligationId:p.id,obligationName:i===0?'Интернет':'Аренда',timing:'upcoming'}))}]}
const batch={id:'demo-batch',revision:1,createdAt:'2026-10-09T00:00:00Z',counts:{unresolved:1},domainCounts:{transaction:1},source:{sourceId:'Пример таблицы',snapshotVersion:1}}
const row={id:'demo-record',source:{sheet:'Операции',row:5,domain:'transaction',raw:{},normalized:{kind:'purchase',amountMinor:128000,currency:'RUB',description:'Продукты',date:{local:'2026-10-09',precision:'date'}},issues:[]},decision:{action:'unresolved'},candidates:[],history:[],published:false}
;(apiClient as unknown as {GET:(p:string)=>Promise<unknown>}).GET=async(p:string)=>{
 let data:unknown={items:[]}
 if(p.endsWith('/accounts'))data={items:accounts}
 else if(p.endsWith('/categories'))data={items:categories}
 else if(p.endsWith('/financial-events'))data={items:events}
 else if(p.endsWith('/obligations'))data={items:obligations}
 else if(p.endsWith('/cash-position'))data=cash
 else if(p.endsWith('/statistics'))data={from:'2026-05-01',to:'2026-10-10',granularity:'month',coverage:'recorded_only',currencies:[item]}
 else if(p.endsWith('/period-summary'))data={from:'2026-05-01',to:'2026-10-10',currencies:[item.summary]}
 else if(p==='/api/auth/profile')data={id:'demo-user',displayName:'Анна',firstName:'Анна',lastName:'',email:'demo@example.test'}
 else if(p==='/api/auth/methods')data={mode:'password',password:true,email:false,telegram:false,emailLinkAvailable:true}
 else if(p==='/api/auth/me')return {response:new Response(null,{status:401})}
 else if(p==='/api/imports/batches')data={items:[batch]}
 else if(p.endsWith('/records/{recordId}'))data=row
 else if(p.endsWith('/records'))data={items:[row]}
 else if(p==='/api/imports/batches/{batchId}')data=batch
 return {data,response:new Response(null,{status:200})}
}
const blocked=async()=>({error:{message:'Демонстрация: данные не сохраняются и письма не отправляются.'},response:new Response(null,{status:503})})
;(apiClient as unknown as {POST:typeof blocked;PATCH:typeof blocked}).POST=blocked
;(apiClient as unknown as {POST:typeof blocked;PATCH:typeof blocked}).PATCH=blocked
window.fetch=async()=>new Response(JSON.stringify({message:'Демонстрация без сетевых действий.'}),{status:503,headers:{'Content-Type':'application/json'}})
const screen=new URLSearchParams(window.location.search).get('screen')??'/'
createRoot(document.getElementById('root')!).render(<ThemeProvider attribute="class" defaultTheme="light" enableSystem={false}><MemoryRouter initialEntries={[screen]}><QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><AuthContext.Provider value={{session:{user:{id:'demo-user',name:'Анна',role:'admin'},csrfToken:'demo',mode:'password'},logout:async()=>{},revokeSessions:async()=>{}}}><div className="bg-accent px-4 py-2 text-center text-xs text-accent-foreground lg:ml-60">Визуальный пример · суммы фиксированы · запись и отправка писем отключены</div><Routes><Route path="/imports" element={<ImportsPage/>}/><Route path="/assistant" element={<AssistantPage/>}/><Route path="/login" element={<LoginPage/>}/><Route path="/register" element={<PasswordEntry view="register"/>}/><Route path="/forgot-password" element={<PasswordEntry view="forgot"/>}/><Route path="/reset-password" element={<PasswordEntry view="reset"/>}/><Route path="*" element={<HomePage/>}/></Routes></AuthContext.Provider></QueryClientProvider></MemoryRouter></ThemeProvider>)
