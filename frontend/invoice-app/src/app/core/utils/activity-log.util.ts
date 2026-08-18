import { ActivityLog } from '../models/api.model';

export const EVENT_LABELS: Record<string, string> = {
  'auth.login': 'Login realizado',
  'auth.login_failed': 'Falha no login',
  'product.created': 'Produto criado',
  'product.updated': 'Produto atualizado',
  'product.stock_consumed': 'Estoque baixado',
  'product.stock_insufficient': 'Estoque insuficiente',
  'product.stock_not_found': 'Produto não encontrado',
  'product.ai_description': 'Descrição com IA',
  'invoice.created': 'Nota fiscal criada',
  'invoice.print_success': 'Nota fiscal impressa',
  'invoice.print_rejected': 'Impressão bloqueada',
  'invoice.print_error': 'Erro na impressão',
};

export function eventLabel(event: string): string {
  return EVENT_LABELS[event] ?? event;
}

export function isErrorEvent(event: string): boolean {
  return (
    event.includes('error') ||
    event.includes('failed') ||
    event.includes('insufficient') ||
    event.includes('rejected')
  );
}

export function buildSummary(log: ActivityLog): string {
  const d = log.details;
  if (!d) {
    return '';
  }
  switch (log.event) {
    case 'product.created':
    case 'product.updated':
      return `Código ${d['code']} · ${d['description']} · saldo ${d['stock_quantity']}`;
    case 'product.stock_consumed':
    case 'product.stock_insufficient':
    case 'product.stock_not_found':
      return itemsSummary(d['items']);
    case 'product.ai_description':
      return `Código ${d['code']}${d['fallback'] ? ' (fallback offline)' : ''}`;
    case 'invoice.created':
    case 'invoice.print_success':
    case 'invoice.print_rejected':
    case 'invoice.print_error':
      return invoiceSummary(d);
    case 'auth.login':
    case 'auth.login_failed':
      return `Usuário ${d['username']}`;
    default:
      return JSON.stringify(d);
  }
}

export function itemsSummary(items: unknown): string {
  if (!Array.isArray(items)) {
    return '';
  }
  return items
    .map((it) => {
      const i = it as { code?: string; quantity?: number };
      return `${i['code'] ?? '?'} × ${i['quantity'] ?? '?'}`;
    })
    .join(', ');
}

function invoiceSummary(d: Record<string, unknown>): string {
  const number = d['number'] ?? '';
  const parts = [`Nota #${number}`];
  if (d['reason']) {
    parts.push(`· ${reasonLabel(String(d['reason']))}`);
  }
  if (Array.isArray(d['items']) && d['items'].length) {
    parts.push(`(${itemsSummary(d['items'])})`);
  }
  if (d['status']) {
    parts.push(`· status ${d['status']}`);
  }
  return parts.join(' ');
}

function reasonLabel(reason: string): string {
  const labels: Record<string, string> = {
    insufficient_stock: 'estoque insuficiente',
    product_not_found: 'produto não encontrado',
    estoque_unavailable: 'serviço de estoque indisponível',
  };
  return labels[reason] ?? reason;
}