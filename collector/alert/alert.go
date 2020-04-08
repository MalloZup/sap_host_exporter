package alert

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/SUSE/sap_host_exporter/collector"
	"github.com/SUSE/sap_host_exporter/internal/sapcontrol"
)

func NewCollector(webService sapcontrol.WebService) (*alertCollector, error) {

	c := &alertCollector{
		collector.NewDefaultCollector("alert"),
		webService,
	}

	c.SetDescriptor("ha_check", "High Availability system configuration and status checks", []string{"description", "category", "comment"})
	c.SetDescriptor("ha_failover_active", "Whether or not High Availability Failover is active", nil)

	return c, nil
}

type alertCollector struct {
	collector.DefaultCollector
	webService sapcontrol.WebService
}

func (c *alertCollector) Collect(ch chan<- prometheus.Metric) {
	log.Debugln("Collecting Alert metrics")

	errs := collector.RecordConcurrently([]func(ch chan<- prometheus.Metric) error{
		c.recordHAConfigChecks,
		c.recordHAFailoverConfigChecks,
		c.recordHAFailoverActive,
	}, ch)

	for _, err := range errs {
		log.Warnf("Alert Collector scrape failed: %s", err)
	}
}

func (c *alertCollector) recordHAConfigChecks(ch chan<- prometheus.Metric) error {
	response, err := c.webService.HACheckConfig()
	if err != nil {
		return errors.Wrap(err, "SAPControl web service error")
	}

	err = c.recordHAChecks(response.Checks, ch)
	if err != nil {
		return err
	}

	return nil
}

func (c *alertCollector) recordHAFailoverConfigChecks(ch chan<- prometheus.Metric) error {
	response, err := c.webService.HACheckFailoverConfig()

	if err != nil {
		return errors.Wrap(err, "SAPControl web service error")
	}

	err = c.recordHAChecks(response.Checks, ch)
	if err != nil {
		return errors.Wrap(err, "could not record HACheck")
	}

	return nil
}

func (c *alertCollector) recordHAChecks(checks []*sapcontrol.HACheck, ch chan<- prometheus.Metric) error {
	for _, check := range checks {
		err := c.recordHACheck(check, ch)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *alertCollector) recordHACheck(check *sapcontrol.HACheck, ch chan<- prometheus.Metric) error {
	stateCode, err := sapcontrol.HaVerificationStateToFloat(check.State)
	category, err := sapcontrol.HaCheckCategoryToString(check.Category)
	if err != nil {
		return errors.Wrapf(err, "unable to process SAPControl HACheck data: %v", *check)
	}
	ch <- c.MakeGaugeMetric("ha_check", stateCode, check.Description, category, check.Comment)

	return nil
}

func (c *alertCollector) recordHAFailoverActive(ch chan<- prometheus.Metric) error {
	response, err := c.webService.HAGetFailoverConfig()

	if err != nil {
		return errors.Wrap(err, "SAPControl web service error")
	}

	var haActive float64
	if response.HAActive {
		haActive = 1
	}
	ch <- c.MakeGaugeMetric("ha_failover_active", haActive)

	return nil
}