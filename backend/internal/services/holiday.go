package services

import (
	"time"

	"github.com/6tail/lunar-go/HolidayUtil"
	"github.com/6tail/lunar-go/calendar"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/at"
	"github.com/rickar/cal/v2/au"
	"github.com/rickar/cal/v2/be"
	"github.com/rickar/cal/v2/br"
	"github.com/rickar/cal/v2/ca"
	"github.com/rickar/cal/v2/ch"
	"github.com/rickar/cal/v2/de"
	"github.com/rickar/cal/v2/dk"
	"github.com/rickar/cal/v2/es"
	"github.com/rickar/cal/v2/fi"
	"github.com/rickar/cal/v2/fr"
	"github.com/rickar/cal/v2/gb"
	"github.com/rickar/cal/v2/ie"
	"github.com/rickar/cal/v2/it"
	"github.com/rickar/cal/v2/jp"
	"github.com/rickar/cal/v2/nl"
	"github.com/rickar/cal/v2/no"
	"github.com/rickar/cal/v2/nz"
	"github.com/rickar/cal/v2/pl"
	"github.com/rickar/cal/v2/pt"
	"github.com/rickar/cal/v2/se"
	"github.com/rickar/cal/v2/us"
)

type HolidayService struct {
	calendars map[string]*cal.BusinessCalendar
}

func NewHolidayService() *HolidayService {
	s := &HolidayService{
		calendars: make(map[string]*cal.BusinessCalendar),
	}
	s.initCalendars()
	return s
}

func (s *HolidayService) initCalendars() {
	s.calendars["US"] = s.createCalendar("United States", us.Holidays...)
	s.calendars["GB"] = s.createCalendar("United Kingdom", gb.Holidays...)
	s.calendars["DE"] = s.createCalendar("Germany", de.Holidays...)
	s.calendars["FR"] = s.createCalendar("France", fr.Holidays...)
	s.calendars["JP"] = s.createCalendar("Japan", jp.Holidays...)
	s.calendars["AU"] = s.createCalendar("Australia", au.HolidaysNSW...)
	s.calendars["CA"] = s.createCalendar("Canada", ca.Holidays...)
	s.calendars["NZ"] = s.createCalendar("New Zealand", nz.Holidays...)
	s.calendars["IT"] = s.createCalendar("Italy", it.Holidays...)
	s.calendars["ES"] = s.createCalendar("Spain", es.Holidays...)
	s.calendars["NL"] = s.createCalendar("Netherlands", nl.Holidays...)
	s.calendars["BE"] = s.createCalendar("Belgium", be.Holidays...)
	s.calendars["AT"] = s.createCalendar("Austria", at.Holidays...)
	s.calendars["CH"] = s.createCalendar("Switzerland", ch.Holidays...)
	s.calendars["SE"] = s.createCalendar("Sweden", se.Holidays...)
	s.calendars["NO"] = s.createCalendar("Norway", no.Holidays...)
	s.calendars["DK"] = s.createCalendar("Denmark", dk.Holidays...)
	s.calendars["FI"] = s.createCalendar("Finland", fi.Holidays...)
	s.calendars["PL"] = s.createCalendar("Poland", pl.Holidays...)
	s.calendars["PT"] = s.createCalendar("Portugal", pt.Holidays...)
	s.calendars["IE"] = s.createCalendar("Ireland", ie.Holidays...)
	s.calendars["BR"] = s.createCalendar("Brazil", br.Holidays...)
}

func (s *HolidayService) createCalendar(name string, holidays ...*cal.Holiday) *cal.BusinessCalendar {
	c := cal.NewBusinessCalendar()
	c.Name = name
	c.AddHoliday(holidays...)
	return c
}

func (s *HolidayService) IsWorkday(t time.Time, countryCode string) bool {
	if countryCode == "CN" {
		return s.isWorkdayChina(t)
	}

	if countryCode == "NONE" {
		return !cal.IsWeekend(t)
	}

	c, ok := s.calendars[countryCode]
	if !ok {
		return !cal.IsWeekend(t)
	}

	return c.IsWorkday(t)
}

func (s *HolidayService) isWorkdayChina(t time.Time) bool {
	solar := calendar.NewSolarFromDate(t)
	holiday := HolidayUtil.GetHolidayByYmd(solar.GetYear(), solar.GetMonth(), solar.GetDay())

	if holiday != nil {
		return holiday.IsWork()
	}

	weekday := t.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

func (s *HolidayService) IsHoliday(t time.Time, countryCode string) bool {
	return !s.IsWorkday(t, countryCode)
}

func (s *HolidayService) GetSupportedCountries() []CountryInfo {
	countries := []CountryInfo{
		{Code: "CN", Name: "China", NameZh: "中国"},
		{Code: "US", Name: "United States", NameZh: "美国"},
		{Code: "GB", Name: "United Kingdom", NameZh: "英国"},
		{Code: "DE", Name: "Germany", NameZh: "德国"},
		{Code: "FR", Name: "France", NameZh: "法国"},
		{Code: "JP", Name: "Japan", NameZh: "日本"},
		{Code: "AU", Name: "Australia", NameZh: "澳大利亚"},
		{Code: "CA", Name: "Canada", NameZh: "加拿大"},
		{Code: "NZ", Name: "New Zealand", NameZh: "新西兰"},
		{Code: "IT", Name: "Italy", NameZh: "意大利"},
		{Code: "ES", Name: "Spain", NameZh: "西班牙"},
		{Code: "NL", Name: "Netherlands", NameZh: "荷兰"},
		{Code: "BE", Name: "Belgium", NameZh: "比利时"},
		{Code: "AT", Name: "Austria", NameZh: "奥地利"},
		{Code: "CH", Name: "Switzerland", NameZh: "瑞士"},
		{Code: "SE", Name: "Sweden", NameZh: "瑞典"},
		{Code: "NO", Name: "Norway", NameZh: "挪威"},
		{Code: "DK", Name: "Denmark", NameZh: "丹麦"},
		{Code: "FI", Name: "Finland", NameZh: "芬兰"},
		{Code: "PL", Name: "Poland", NameZh: "波兰"},
		{Code: "PT", Name: "Portugal", NameZh: "葡萄牙"},
		{Code: "IE", Name: "Ireland", NameZh: "爱尔兰"},
		{Code: "BR", Name: "Brazil", NameZh: "巴西"},
		{Code: "NONE", Name: "Weekdays Only (Mon-Fri)", NameZh: "仅工作日(周一至周五)"},
	}
	return countries
}

type CountryInfo struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	NameZh string `json:"name_zh"`
}
