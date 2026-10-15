// Grace-specific architecture rules (SPEC-0048).

const APP_TIME_ZONE_PATTERN = /Asia\/Yekaterinburg|\+05:00/

function report(context, node, messageId) {
  context.report({ node, messageId })
}

/** React Query keys belong to entity query factories in entities/<slice>/api. */
const noInlineQueryKey = {
  meta: {
    type: 'problem',
    messages: {
      inline: 'Declare query keys in an entity query factory (entities/<slice>/api), not inline.',
    },
    schema: [],
  },
  create(context) {
    return {
      'Property[key.name="queryKey"] > ArrayExpression'(node) {
        report(context, node, 'inline')
      },
    }
  },
}

/** Cache invalidation belongs to mutation hooks in features/<slice>/api or entities. */
const noDirectInvalidate = {
  meta: {
    type: 'problem',
    messages: {
      direct: 'Invalidate queries inside a mutation hook (features/<slice>/api) or entity api.',
    },
    schema: [],
  },
  create(context) {
    return {
      'CallExpression[callee.property.name="invalidateQueries"]'(node) {
        report(context, node, 'direct')
      },
    }
  },
}

/** The display time zone is a single constant in shared/config. */
const noTimeZoneLiteral = {
  meta: {
    type: 'problem',
    messages: {
      literal: 'Use APP_TIME_ZONE from shared/config instead of a time zone literal.',
    },
    schema: [],
  },
  create(context) {
    return {
      Literal(node) {
        if (typeof node.value === 'string' && APP_TIME_ZONE_PATTERN.test(node.value)) {
          report(context, node, 'literal')
        }
      },
      TemplateElement(node) {
        if (APP_TIME_ZONE_PATTERN.test(node.value.raw)) report(context, node, 'literal')
      },
    }
  },
}

export default {
  meta: { name: 'grace' },
  rules: {
    'no-inline-query-key': noInlineQueryKey,
    'no-direct-invalidate': noDirectInvalidate,
    'no-time-zone-literal': noTimeZoneLiteral,
  },
}
