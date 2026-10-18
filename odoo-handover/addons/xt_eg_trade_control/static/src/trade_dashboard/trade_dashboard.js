/** @odoo-module **/

import { Component, onWillStart, useState } from "@odoo/owl";
import { registry } from "@web/core/registry";
import { useService } from "@web/core/utils/hooks";
import { standardActionServiceProps } from "@web/webclient/actions/action_service";

function isArabic() {
    const lang = (
        window.odoo?.session_info?.user_context?.lang ||
        window.odoo?.session_info?.lang ||
        document.documentElement.lang ||
        navigator.language ||
        ""
    ).toLowerCase();
    return lang.startsWith("ar") || document.documentElement.dir === "rtl" || document.body?.classList.contains("o_rtl");
}

function label(en, ar) {
    return isArabic() ? ar : en;
}

export class TradeControlDashboard extends Component {
    static template = "xt_eg_trade_control.TradeControlDashboard";
    static props = { ...standardActionServiceProps };

    setup() {
        this.orm = useService("orm");
        this.action = useService("action");
        this.state = useState({ loading: true, error: "", data: {} });
        onWillStart(() => this.load());
    }

    async load() {
        this.state.loading = true;
        this.state.error = "";
        try {
            this.state.data = await this.orm.call("eg.trade.dashboard", "get_dashboard_data", [], {});
        } catch (error) {
            this.state.error = error?.data?.message || error.message || this.ui("Could not load dashboard.", "تعذر تحميل اللوحة.");
        } finally {
            this.state.loading = false;
        }
    }

    ui(en, ar) {
        return label(en, ar);
    }

    dir() {
        return isArabic() ? "rtl" : "ltr";
    }

    amount(value) {
        const data = this.state.data || {};
        const formatted = new Intl.NumberFormat(isArabic() ? "ar-EG" : "en-US", {
            maximumFractionDigits: 2,
        }).format(Number(value || 0));
        return data.currency_position === "before"
            ? `${data.currency_symbol || ""} ${formatted}`
            : `${formatted} ${data.currency_symbol || ""}`;
    }

    count(value) {
        return new Intl.NumberFormat(isArabic() ? "ar-EG" : "en-US").format(Number(value || 0));
    }

    ratio(value, max) {
        return Math.max(4, Math.min(100, Math.round((Number(value || 0) / Math.max(Number(max || 0), 1)) * 100)));
    }

    maxSales() {
        return Math.max(...(this.state.data.sales_series || []).map((item) => Number(item.sales || 0)), 1);
    }

    etaCount(state) {
        return Number((this.state.data.eta || {})[state] || 0);
    }

    openAction(xmlId) {
        this.action.doAction(xmlId);
    }

    cards() {
        const d = this.state.data;
        return [
            { key: "today", title: this.ui("Today's Sales", "مبيعات اليوم"), value: this.amount(d.today_sales), hint: this.ui("Confirmed orders", "أوامر مؤكدة"), icon: "fa-shopping-cart", tone: "teal", action: "xt_eg_trade_control.action_eg_trade_sale_orders" },
            { key: "cash", title: this.ui("Collected Today", "تحصيل اليوم"), value: this.amount(d.today_collected), hint: this.ui("Paid invoices", "فواتير مسددة"), icon: "fa-money", tone: "green", action: "xt_eg_trade_control.action_eg_trade_invoices" },
            { key: "deliveries", title: this.ui("Pending Delivery", "تسليمات مفتوحة"), value: this.count(d.pending_deliveries), hint: this.ui("Need warehouse action", "تحتاج المخزن"), icon: "fa-truck", tone: "blue", action: "xt_eg_trade_control.action_eg_trade_deliveries" },
            { key: "stock", title: this.ui("Low Stock", "نواقص المخزن"), value: this.count(d.low_stock), hint: this.ui("Below simple reorder point", "تحت حد الطلب"), icon: "fa-cubes", tone: "amber", action: "xt_eg_trade_control.action_eg_trade_products" },
            { key: "tax", title: this.ui("Rejected Tax Docs", "مستندات مرفوضة"), value: this.count(this.etaCount("rejected")), hint: this.ui("Need accountant review", "تحتاج مراجعة"), icon: "fa-exclamation-triangle", tone: "red", action: "xt_eg_trade_control.action_eg_trade_eta_document" },
        ];
    }

    quickActions() {
        return [
            { key: "easy", title: this.ui("Easy Sale", "بيع سهل"), icon: "fa-bolt", action: "xt_eg_trade_control.action_eg_trade_easy_sale" },
            { key: "customer", title: this.ui("Customer", "عميل"), icon: "fa-user", action: "xt_eg_trade_control.action_eg_trade_customers" },
            { key: "product", title: this.ui("Product", "صنف"), icon: "fa-tag", action: "xt_eg_trade_control.action_eg_trade_products" },
            { key: "invoice", title: this.ui("Invoice", "فاتورة"), icon: "fa-file-text-o", action: "xt_eg_trade_control.action_eg_trade_invoices" },
            { key: "stock", title: this.ui("Warehouse", "مخزن"), icon: "fa-archive", action: "xt_eg_trade_control.action_eg_trade_stock_moves" },
            { key: "tax", title: this.ui("Tax Follow-up", "متابعة الضريبة"), icon: "fa-shield", action: "xt_eg_trade_control.action_eg_trade_eta_document" },
        ];
    }
}

registry.category("actions").add("trade_control_dashboard", TradeControlDashboard);
